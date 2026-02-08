/**
 * ServiceManager - Centralized service orchestration for MycelicMemory
 * Auto-starts backend, Ollama, and Qdrant when the desktop app launches.
 */

import { ChildProcess, spawn, execSync } from 'child_process';
import { BrowserWindow } from 'electron';
import * as path from 'path';
import * as fs from 'fs';
import type { AppSettings } from '../../shared/types';

export interface ServiceStatus {
  backend: { running: boolean; port?: number; version?: string; managedByUs: boolean };
  ollama: { running: boolean; version?: string; models?: string[]; missingModels?: string[]; managedByUs: boolean };
  qdrant: { running: boolean; version?: string; managedByUs: boolean };
}

export class ServiceManager {
  private backendProcess: ChildProcess | null = null;
  private ollamaProcess: ChildProcess | null = null;
  private qdrantContainerId: string | null = null;
  private settings: AppSettings;
  private statusInterval: NodeJS.Timeout | null = null;
  private backendManagedByUs = false;
  private ollamaManagedByUs = false;
  private qdrantManagedByUs = false;

  constructor(settings: AppSettings) {
    this.settings = settings;
  }

  /** Auto-start all services on Electron launch */
  async ensureAllServices(): Promise<ServiceStatus> {
    console.log('[ServiceManager] Ensuring all services are running...');

    // Start backend first (other services don't depend on it, but the app does)
    await this.ensureBackend();

    // Start Ollama and Qdrant in parallel
    await Promise.all([
      this.ensureOllama().catch(err => console.warn('[ServiceManager] Ollama auto-start failed:', err.message)),
      this.ensureQdrant().catch(err => console.warn('[ServiceManager] Qdrant auto-start failed:', err.message)),
    ]);

    return this.getFullStatus();
  }

  // ── Backend ──────────────────────────────────────────────

  async isBackendRunning(): Promise<boolean> {
    try {
      const resp = await fetchWithTimeout(
        `http://127.0.0.1:${this.settings.api_port}/api/v1/health`,
        3000
      );
      return resp.ok;
    } catch {
      return false;
    }
  }

  async ensureBackend(): Promise<boolean> {
    if (await this.isBackendRunning()) {
      console.log('[ServiceManager] Backend already running on port', this.settings.api_port);
      return true;
    }

    console.log('[ServiceManager] Backend not running, attempting auto-start...');
    return this.startBackend();
  }

  async startBackend(): Promise<boolean> {
    // Check if already running first (avoids daemon "already running" error)
    if (await this.isBackendRunning()) {
      console.log('[ServiceManager] Backend is already running');
      return true;
    }

    const binaryPath = this.findBackendBinary();
    if (!binaryPath) {
      console.warn('[ServiceManager] Could not find mycelicmemory binary. Searched: CWD, parent dir, app dir, resources, PATH, common install locations.');
      return false;
    }

    console.log('[ServiceManager] Starting backend:', binaryPath);

    try {
      // First, try to clean up any stale daemon state by running 'stop'
      try {
        execSync(`"${binaryPath}" stop`, { timeout: 3000, windowsHide: true, stdio: 'ignore' });
      } catch {
        // Expected to fail if daemon isn't running — ignore
      }

      this.backendProcess = spawn(binaryPath, ['start', '--port', String(this.settings.api_port)], {
        detached: true,
        stdio: 'ignore',
        windowsHide: true,
      });

      this.backendProcess.unref();
      this.backendManagedByUs = true;

      // Poll for backend to come up
      const started = await pollUntilReady(
        () => this.isBackendRunning(),
        10000,
        500,
      );

      if (!started) {
        console.warn('[ServiceManager] Backend failed to start within 10s');
      }
      return started;
    } catch (err: any) {
      console.error('[ServiceManager] Failed to start backend:', err.message);
      return false;
    }
  }

  async stopBackend(): Promise<void> {
    if (!this.backendManagedByUs) return;

    const binaryPath = this.findBackendBinary();
    if (binaryPath) {
      try {
        execSync(`"${binaryPath}" stop`, { timeout: 5000, windowsHide: true });
      } catch {
        // If stop command fails, kill the process directly
        if (this.backendProcess && !this.backendProcess.killed) {
          this.backendProcess.kill();
        }
      }
    }
    this.backendProcess = null;
    this.backendManagedByUs = false;
  }

  private findBackendBinary(): string | null {
    const isWin = process.platform === 'win32';
    const binaryName = isWin ? 'mycelicmemory.exe' : 'mycelicmemory';

    // 1. Check working directory (dev mode)
    const cwdPath = path.join(process.cwd(), binaryName);
    if (fs.existsSync(cwdPath)) return cwdPath;

    // 1b. Check parent directory (dev mode: Electron runs from desktop/)
    const parentPath = path.join(process.cwd(), '..', binaryName);
    if (fs.existsSync(parentPath)) return path.resolve(parentPath);

    // 2. Check next to the Electron app
    const appDir = path.dirname(process.execPath);
    const appPath = path.join(appDir, binaryName);
    if (fs.existsSync(appPath)) return appPath;

    // 3. Check app resources
    try {
      const resourcesPath = path.join(process.resourcesPath, binaryName);
      if (fs.existsSync(resourcesPath)) return resourcesPath;
    } catch { /* resourcesPath may not exist in dev */ }

    // 4. Check PATH
    try {
      const cmd = isWin ? 'where' : 'which';
      const result = execSync(`${cmd} ${binaryName}`, { timeout: 3000, windowsHide: true, encoding: 'utf-8' });
      const found = result.trim().split('\n')[0].trim();
      if (found && fs.existsSync(found)) return found;
    } catch { /* not in PATH */ }

    // 5. Check common install locations
    const home = process.env.HOME || process.env.USERPROFILE || '';
    const candidates = isWin
      ? [
          path.join(home, '.mycelicmemory', 'bin', binaryName),
          path.join(process.env.LOCALAPPDATA || '', 'mycelicmemory', binaryName),
        ]
      : [
          '/usr/local/bin/' + binaryName,
          path.join(home, '.local', 'bin', binaryName),
          path.join(home, '.mycelicmemory', 'bin', binaryName),
        ];

    for (const candidate of candidates) {
      if (candidate && fs.existsSync(candidate)) return candidate;
    }

    return null;
  }

  // ── Ollama ───────────────────────────────────────────────

  async checkOllama(): Promise<{ available: boolean; version?: string; models?: string[]; missingModels?: string[] }> {
    try {
      const baseUrl = this.settings.ollama_base_url;

      // Check availability
      const tagsResp = await fetchWithTimeout(`${baseUrl}/api/tags`, 5000);
      if (!tagsResp.ok) return { available: false };

      const tagsData = await tagsResp.json();
      const models = (tagsData.models || []).map((m: any) => m.name);
      const modelSet = new Set<string>();
      for (const m of models) {
        modelSet.add(m);
        modelSet.add(m.split(':')[0]); // base name without tag
      }

      // Check required models
      const required = [this.settings.ollama_embedding_model, this.settings.ollama_chat_model];
      const missingModels = required.filter(m => !modelSet.has(m) && !modelSet.has(m.split(':')[0]));

      // Get version
      let version: string | undefined;
      try {
        const vResp = await fetchWithTimeout(`${baseUrl}/api/version`, 3000);
        if (vResp.ok) {
          const vData = await vResp.json();
          version = vData.version;
        }
      } catch { /* ignore version check failure */ }

      return { available: true, version, models, missingModels };
    } catch {
      return { available: false };
    }
  }

  async ensureOllama(): Promise<boolean> {
    const status = await this.checkOllama();
    if (status.available) {
      console.log('[ServiceManager] Ollama already running');
      // Pull missing models silently
      if (status.missingModels && status.missingModels.length > 0) {
        this.pullOllamaModels(status.missingModels);
      }
      return true;
    }

    console.log('[ServiceManager] Ollama not running, attempting auto-start...');
    return this.startOllama();
  }

  async startOllama(): Promise<boolean> {
    try {
      const isWin = process.platform === 'win32';

      // Try to find ollama binary
      let ollamaBin: string | null = null;
      try {
        const cmd = isWin ? 'where' : 'which';
        const result = execSync(`${cmd} ollama`, { timeout: 3000, windowsHide: true, encoding: 'utf-8' });
        ollamaBin = result.trim().split('\n')[0].trim();
      } catch { /* not in PATH */ }

      if (!ollamaBin) {
        // On Windows, check common install location
        if (isWin) {
          const candidates = [
            path.join(process.env.LOCALAPPDATA || '', 'Programs', 'Ollama', 'ollama.exe'),
            path.join(process.env.PROGRAMFILES || '', 'Ollama', 'ollama.exe'),
          ];
          for (const c of candidates) {
            if (fs.existsSync(c)) { ollamaBin = c; break; }
          }
        }
      }

      if (!ollamaBin) {
        console.warn('[ServiceManager] Ollama binary not found');
        return false;
      }

      console.log('[ServiceManager] Starting Ollama:', ollamaBin);

      this.ollamaProcess = spawn(ollamaBin, ['serve'], {
        detached: true,
        stdio: 'ignore',
        windowsHide: true,
      });
      this.ollamaProcess.unref();
      this.ollamaManagedByUs = true;

      // Poll for Ollama to come up
      const ready = await pollUntilReady(
        async () => (await this.checkOllama()).available,
        15000,
        1000,
      );

      if (ready) {
        // Pull missing models after startup
        const status = await this.checkOllama();
        if (status.missingModels && status.missingModels.length > 0) {
          this.pullOllamaModels(status.missingModels);
        }
      }

      return ready;
    } catch (err: any) {
      console.error('[ServiceManager] Failed to start Ollama:', err.message);
      return false;
    }
  }

  private pullOllamaModels(models: string[]): void {
    for (const model of models) {
      console.log('[ServiceManager] Pulling Ollama model:', model);
      try {
        // Fire and forget - models pull in background
        const proc = spawn('ollama', ['pull', model], {
          detached: true,
          stdio: 'ignore',
          windowsHide: true,
        });
        proc.unref();
      } catch (err: any) {
        console.warn('[ServiceManager] Failed to pull model', model, ':', err.message);
      }
    }
  }

  // ── Qdrant ───────────────────────────────────────────────

  async checkQdrant(): Promise<{ available: boolean; version?: string }> {
    try {
      const resp = await fetchWithTimeout(`${this.settings.qdrant_url}/collections`, 5000);
      if (!resp.ok) return { available: false };

      // Get version
      let version: string | undefined;
      try {
        const vResp = await fetchWithTimeout(this.settings.qdrant_url, 3000);
        if (vResp.ok) {
          const vData = await vResp.json();
          version = vData.version;
        }
      } catch { /* ignore */ }

      return { available: true, version };
    } catch {
      return { available: false };
    }
  }

  async ensureQdrant(): Promise<boolean> {
    if (!this.settings.qdrant_enabled) {
      console.log('[ServiceManager] Qdrant disabled in settings');
      return false;
    }

    const status = await this.checkQdrant();
    if (status.available) {
      console.log('[ServiceManager] Qdrant already running');
      return true;
    }

    console.log('[ServiceManager] Qdrant not running, attempting auto-start via Docker...');
    return this.startQdrant();
  }

  async startQdrant(): Promise<boolean> {
    // Check if Docker is available
    if (!this.isDockerAvailable()) {
      console.warn('[ServiceManager] Docker not available, cannot auto-start Qdrant');
      return false;
    }

    try {
      // Check if container already exists (stopped)
      try {
        execSync('docker start mycelicmemory-qdrant', { timeout: 10000, windowsHide: true, stdio: 'ignore' });
        console.log('[ServiceManager] Restarted existing Qdrant container');
      } catch {
        // Container doesn't exist, create new one
        console.log('[ServiceManager] Creating new Qdrant container...');
        const result = execSync(
          'docker run -d --name mycelicmemory-qdrant -p 6333:6333 -v qdrant_storage:/qdrant/storage qdrant/qdrant',
          { timeout: 60000, windowsHide: true, encoding: 'utf-8' }
        );
        this.qdrantContainerId = result.trim();
      }

      this.qdrantManagedByUs = true;

      // Poll for Qdrant to come up
      return await pollUntilReady(
        async () => (await this.checkQdrant()).available,
        15000,
        1000,
      );
    } catch (err: any) {
      console.error('[ServiceManager] Failed to start Qdrant:', err.message);
      return false;
    }
  }

  private isDockerAvailable(): boolean {
    try {
      execSync('docker info', { timeout: 5000, windowsHide: true, stdio: 'ignore' });
      return true;
    } catch {
      return false;
    }
  }

  // ── Status & Polling ─────────────────────────────────────

  async getFullStatus(): Promise<ServiceStatus> {
    const [backendRunning, ollamaStatus, qdrantStatus] = await Promise.all([
      this.isBackendRunning(),
      this.checkOllama(),
      this.checkQdrant(),
    ]);

    return {
      backend: {
        running: backendRunning,
        port: backendRunning ? this.settings.api_port : undefined,
        managedByUs: this.backendManagedByUs,
      },
      ollama: {
        running: ollamaStatus.available,
        version: ollamaStatus.version,
        models: ollamaStatus.models,
        missingModels: ollamaStatus.missingModels,
        managedByUs: this.ollamaManagedByUs,
      },
      qdrant: {
        running: qdrantStatus.available,
        version: qdrantStatus.version,
        managedByUs: this.qdrantManagedByUs,
      },
    };
  }

  startStatusPolling(window: BrowserWindow, intervalMs = 15000): void {
    this.stopStatusPolling();

    const sendStatus = async () => {
      if (window.isDestroyed()) return;
      try {
        const status = await this.getFullStatus();
        window.webContents.send('services:status-update', status);
      } catch { /* window may have closed */ }
    };

    // Send initial status
    sendStatus();

    this.statusInterval = setInterval(sendStatus, intervalMs);
  }

  stopStatusPolling(): void {
    if (this.statusInterval) {
      clearInterval(this.statusInterval);
      this.statusInterval = null;
    }
  }

  // ── Cleanup ──────────────────────────────────────────────

  async cleanup(): Promise<void> {
    console.log('[ServiceManager] Cleaning up...');
    this.stopStatusPolling();

    // Stop backend if we started it
    if (this.backendManagedByUs) {
      await this.stopBackend();
    }

    // We don't stop Ollama or Qdrant on quit - they're shared system services
    // that other tools may be using
  }
}

// ── Helpers ──────────────────────────────────────────────────

async function fetchWithTimeout(url: string, timeoutMs: number): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(url, { signal: controller.signal });
  } finally {
    clearTimeout(timer);
  }
}

async function pollUntilReady(
  check: () => Promise<boolean>,
  timeoutMs: number,
  intervalMs: number,
): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (await check()) return true;
    await new Promise(r => setTimeout(r, intervalMs));
  }
  return false;
}

/**
 * Claude Chat Stream Service Controller
 * Manages the claude-chat-stream daemon from MycelicMemory desktop
 */

import { spawn, ChildProcess, exec } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';
import EventSource from 'eventsource';

export interface ClaudeChatStreamStatus {
  isRunning: boolean;
  pid: number | null;
  uptime: number;
  startedAt: string | null;
  apiPort: number;
  database: {
    messageCount: number;
    sessionCount: number;
    projectCount: number;
    toolCallCount: number;
    fileRefCount: number;
    databaseSizeBytes: number;
  } | null;
  captureStats: {
    totalMessagesIngested: number;
    totalToolCallsIngested: number;
    totalFileRefsIngested: number;
  } | null;
}

export interface SSEEvent {
  type: string;
  data: unknown;
  timestamp: string;
}

export class ClaudeChatStreamService {
  private apiUrl: string;
  private projectPath: string;
  private eventSource: EventSource | null = null;
  private eventListeners: Map<string, Set<(data: unknown) => void>> = new Map();
  private process: ChildProcess | null = null;

  constructor(options: { apiPort?: number; projectPath?: string } = {}) {
    this.apiUrl = `http://localhost:${options.apiPort || 9848}`;
    this.projectPath = options.projectPath || this.findProjectPath();
  }

  private findProjectPath(): string {
    // Try common locations
    const possiblePaths = [
      'C:\\dev\\active\\ai\\claude-chat-stream',
      path.join(process.env.USERPROFILE || '', 'dev', 'claude-chat-stream'),
      path.join(process.env.HOME || '', 'dev', 'claude-chat-stream'),
    ];

    for (const p of possiblePaths) {
      if (fs.existsSync(path.join(p, 'package.json'))) {
        return p;
      }
    }

    return possiblePaths[0]; // Default to first path
  }

  /**
   * Check if the daemon is running by calling the health endpoint
   */
  async isRunning(): Promise<boolean> {
    try {
      const response = await fetch(`${this.apiUrl}/health`, {
        signal: AbortSignal.timeout(2000),
      });
      return response.ok;
    } catch {
      return false;
    }
  }

  /**
   * Get daemon status
   */
  async getStatus(): Promise<ClaudeChatStreamStatus> {
    try {
      const response = await fetch(`${this.apiUrl}/api/v1/daemon/status`, {
        signal: AbortSignal.timeout(5000),
      });

      if (!response.ok) {
        throw new Error(`Status request failed: ${response.statusText}`);
      }

      return await response.json() as ClaudeChatStreamStatus;
    } catch (error) {
      // Daemon not running, return offline status
      return {
        isRunning: false,
        pid: null,
        uptime: 0,
        startedAt: null,
        apiPort: 9848,
        database: null,
        captureStats: null,
      };
    }
  }

  /**
   * Start the daemon by spawning the process
   */
  async start(): Promise<boolean> {
    if (await this.isRunning()) {
      console.log('[ClaudeChatStream] Daemon already running');
      return true;
    }

    return new Promise((resolve) => {
      try {
        // Use npx tsx to run the CLI
        const npmPath = process.platform === 'win32' ? 'npx.cmd' : 'npx';

        this.process = spawn(npmPath, ['tsx', 'src/cli/index.ts', 'start'], {
          cwd: this.projectPath,
          detached: true,
          stdio: 'ignore',
          shell: true,
          windowsHide: true,
        });

        this.process.unref();

        // Wait for the daemon to start
        let attempts = 0;
        const checkInterval = setInterval(async () => {
          attempts++;
          if (await this.isRunning()) {
            clearInterval(checkInterval);
            console.log('[ClaudeChatStream] Daemon started successfully');
            resolve(true);
          } else if (attempts >= 20) {
            clearInterval(checkInterval);
            console.error('[ClaudeChatStream] Daemon failed to start');
            resolve(false);
          }
        }, 500);
      } catch (error) {
        console.error('[ClaudeChatStream] Failed to start daemon:', error);
        resolve(false);
      }
    });
  }

  /**
   * Stop the daemon
   */
  async stop(): Promise<boolean> {
    try {
      // First try graceful shutdown via CLI
      return new Promise((resolve) => {
        const npmPath = process.platform === 'win32' ? 'npx.cmd' : 'npx';

        exec(`${npmPath} tsx src/cli/index.ts stop`, {
          cwd: this.projectPath,
        }, async (error) => {
          if (error) {
            console.error('[ClaudeChatStream] Stop command failed:', error);
          }

          // Wait and verify
          await new Promise(r => setTimeout(r, 1000));
          const stillRunning = await this.isRunning();
          resolve(!stillRunning);
        });
      });
    } catch (error) {
      console.error('[ClaudeChatStream] Failed to stop daemon:', error);
      return false;
    }
  }

  /**
   * Connect to SSE stream for real-time events
   */
  connectSSE(): void {
    if (this.eventSource) {
      this.disconnectSSE();
    }

    this.eventSource = new EventSource(`${this.apiUrl}/api/v1/events`);

    this.eventSource.onopen = () => {
      console.log('[ClaudeChatStream] SSE connected');
      this.emit('connected', {});
    };

    this.eventSource.onerror = (error) => {
      console.error('[ClaudeChatStream] SSE error:', error);
      this.emit('error', error);
    };

    // Listen for specific event types
    const eventTypes = ['status', 'message', 'session', 'sync', 'change', 'log'];

    eventTypes.forEach(type => {
      this.eventSource!.addEventListener(type, (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data);
          this.emit(type, data);
        } catch (err) {
          console.error(`[ClaudeChatStream] Failed to parse ${type} event:`, err);
        }
      });
    });
  }

  /**
   * Disconnect SSE stream
   */
  disconnectSSE(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
      this.emit('disconnected', {});
    }
  }

  /**
   * Subscribe to events
   */
  on(event: string, callback: (data: unknown) => void): void {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, new Set());
    }
    this.eventListeners.get(event)!.add(callback);
  }

  /**
   * Unsubscribe from events
   */
  off(event: string, callback: (data: unknown) => void): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      listeners.delete(callback);
    }
  }

  /**
   * Emit event to listeners
   */
  private emit(event: string, data: unknown): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      listeners.forEach(callback => {
        try {
          callback(data);
        } catch (err) {
          console.error(`[ClaudeChatStream] Error in ${event} listener:`, err);
        }
      });
    }
  }

  /**
   * Get recent logs from the daemon
   */
  async getLogs(limit = 100, level?: string): Promise<unknown[]> {
    try {
      let url = `${this.apiUrl}/api/v1/logs?limit=${limit}`;
      if (level) url += `&level=${level}`;

      const response = await fetch(url, {
        signal: AbortSignal.timeout(5000),
      });

      if (!response.ok) {
        return [];
      }

      const data = await response.json() as { logs: unknown[] };
      return data.logs || [];
    } catch {
      return [];
    }
  }

  /**
   * Get database stats
   */
  async getStats(): Promise<unknown> {
    try {
      const response = await fetch(`${this.apiUrl}/api/v1/stats`, {
        signal: AbortSignal.timeout(5000),
      });

      if (!response.ok) {
        return null;
      }

      return await response.json();
    } catch {
      return null;
    }
  }
}

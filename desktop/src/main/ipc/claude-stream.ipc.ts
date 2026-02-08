/**
 * Claude Chat Stream IPC Handlers
 * Handles daemon control and real-time streaming from claude-chat-stream
 */

import type { IpcMain, BrowserWindow } from 'electron';
import { ClaudeChatStreamService } from '../services/claude-chat-stream-service';

let service: ClaudeChatStreamService | null = null;
let mainWindow: BrowserWindow | null = null;

export function registerClaudeChatStreamHandlers(
  ipcMain: IpcMain,
  window: BrowserWindow,
  projectPath?: string
): void {
  mainWindow = window;

  // Initialize service
  service = new ClaudeChatStreamService({ projectPath });

  // Set up event forwarding to renderer
  setupEventForwarding();

  // Get daemon status
  ipcMain.handle('claude-stream:status', async () => {
    try {
      return await service!.getStatus();
    } catch (error) {
      console.error('Failed to get claude-stream status:', error);
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
  });

  // Check if running
  ipcMain.handle('claude-stream:is-running', async () => {
    try {
      return await service!.isRunning();
    } catch (error) {
      console.error('Failed to check claude-stream status:', error);
      return false;
    }
  });

  // Start daemon
  ipcMain.handle('claude-stream:start', async () => {
    try {
      const success = await service!.start();
      if (success) {
        // Connect to SSE after daemon starts
        setTimeout(() => {
          service!.connectSSE();
        }, 1000);
      }
      return success;
    } catch (error) {
      console.error('Failed to start claude-stream:', error);
      return false;
    }
  });

  // Stop daemon
  ipcMain.handle('claude-stream:stop', async () => {
    try {
      service!.disconnectSSE();
      return await service!.stop();
    } catch (error) {
      console.error('Failed to stop claude-stream:', error);
      return false;
    }
  });

  // Connect to SSE
  ipcMain.handle('claude-stream:connect-sse', async () => {
    try {
      if (await service!.isRunning()) {
        service!.connectSSE();
        return true;
      }
      return false;
    } catch (error) {
      console.error('Failed to connect SSE:', error);
      return false;
    }
  });

  // Disconnect SSE
  ipcMain.handle('claude-stream:disconnect-sse', async () => {
    try {
      service!.disconnectSSE();
      return true;
    } catch (error) {
      console.error('Failed to disconnect SSE:', error);
      return false;
    }
  });

  // Get logs
  ipcMain.handle('claude-stream:logs', async (_event, limit?: number, level?: string) => {
    try {
      return await service!.getLogs(limit, level);
    } catch (error) {
      console.error('Failed to get claude-stream logs:', error);
      return [];
    }
  });

  // Get stats
  ipcMain.handle('claude-stream:stats', async () => {
    try {
      return await service!.getStats();
    } catch (error) {
      console.error('Failed to get claude-stream stats:', error);
      return null;
    }
  });

  // Auto-connect to SSE if daemon is running
  autoConnectSSE();
}

function setupEventForwarding(): void {
  if (!service || !mainWindow) return;

  const eventTypes = ['connected', 'disconnected', 'error', 'status', 'message', 'session', 'sync', 'change', 'log'];

  eventTypes.forEach(type => {
    service!.on(type, (data) => {
      if (mainWindow && !mainWindow.isDestroyed()) {
        mainWindow.webContents.send(`claude-stream:event:${type}`, data);
      }
    });
  });
}

async function autoConnectSSE(): Promise<void> {
  if (!service) return;

  // Check if daemon is running and connect
  const isRunning = await service.isRunning();
  if (isRunning) {
    console.log('[ClaudeStreamIPC] Daemon running, connecting SSE');
    service.connectSSE();
  } else {
    console.log('[ClaudeStreamIPC] Daemon not running');
  }
}

export function getClaudeChatStreamService(): ClaudeChatStreamService | null {
  return service;
}

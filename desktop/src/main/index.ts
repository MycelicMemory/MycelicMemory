/**
 * MycelicMemory Desktop - Electron Main Process
 * Entry point for the Electron application
 */

import { app, BrowserWindow, ipcMain, shell } from 'electron';
import * as path from 'path';
import Store from 'electron-store';
import { registerMemoryHandlers } from './ipc/memory.ipc';
import { registerClaudeHandlers } from './ipc/claude.ipc';
import { registerConfigHandlers } from './ipc/config.ipc';
import { registerClaudeChatStreamHandlers } from './ipc/claude-stream.ipc';
import { MycelicMemoryClient } from './services/mycelicmemory-client';
import { ServiceManager } from './services/service-manager';
import { registerServicesHandlers } from './ipc/services.ipc';
import { AppSettings } from '../shared/types';

// Initialize electron-store for settings persistence
const store = new Store<{ settings: AppSettings }>({
  defaults: {
    settings: {
      api_url: 'http://127.0.0.1',
      api_port: 3099,
      ollama_base_url: 'http://127.0.0.1:11434',
      ollama_embedding_model: 'nomic-embed-text',
      ollama_chat_model: 'llama3.2',
      qdrant_url: 'http://127.0.0.1:6333',
      qdrant_enabled: true,
      claude_stream_db_path: getDefaultClaudeStreamDbPath(),
      theme: 'dark',
      sidebar_collapsed: false,
    },
  },
});

let mainWindow: BrowserWindow | null = null;
let serviceManager: ServiceManager | null = null;

function getDefaultClaudeStreamDbPath(): string {
  const platform = process.platform;
  const home = app.getPath('home');

  if (platform === 'win32') {
    return path.join(process.env.LOCALAPPDATA || path.join(home, 'AppData', 'Local'), 'claude-chat-stream', 'data', 'chats.db');
  } else if (platform === 'darwin') {
    return path.join(home, 'Library', 'Application Support', 'claude-chat-stream', 'data', 'chats.db');
  } else {
    return path.join(home, '.config', 'claude-chat-stream', 'data', 'chats.db');
  }
}

function createWindow(): void {
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 1000,
    minHeight: 700,
    frame: true,
    titleBarStyle: process.platform === 'darwin' ? 'hiddenInset' : 'default',
    backgroundColor: '#0f172a',
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, 'preload.js'),
    },
  });

  // In development, load from Vite dev server
  if (process.env.NODE_ENV === 'development') {
    mainWindow.loadURL('http://localhost:5173');
    mainWindow.webContents.openDevTools();
  } else {
    // In production, load from built files
    mainWindow.loadFile(path.join(__dirname, '../renderer/index.html'));
  }

  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  // Open external links in default browser
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url);
    return { action: 'deny' };
  });
}

function initializeServicesAndHandlers(): void {
  const settings = store.get('settings');

  // Normalize localhost → 127.0.0.1 to avoid IPv6 resolution issues on Windows
  // (Node.js fetch can resolve 'localhost' to ::1 which may fail)
  if (settings.api_url.includes('localhost')) {
    settings.api_url = settings.api_url.replace('localhost', '127.0.0.1');
  }
  if (settings.ollama_base_url.includes('localhost')) {
    settings.ollama_base_url = settings.ollama_base_url.replace('localhost', '127.0.0.1');
  }
  if (settings.qdrant_url.includes('localhost')) {
    settings.qdrant_url = settings.qdrant_url.replace('localhost', '127.0.0.1');
  }

  const apiBaseUrl = `${settings.api_url}:${settings.api_port}`;
  const claudeDbPath = settings.claude_stream_db_path;

  // Create instances immediately (no async work)
  serviceManager = new ServiceManager(settings);

  // Register ALL IPC handlers immediately so the renderer can make calls right away
  const client = new MycelicMemoryClient(apiBaseUrl);
  registerMemoryHandlers(ipcMain, apiBaseUrl);
  registerClaudeHandlers(ipcMain, client);
  registerConfigHandlers(ipcMain, store);
  registerServicesHandlers(ipcMain, serviceManager);

  if (mainWindow) {
    registerClaudeChatStreamHandlers(ipcMain, mainWindow, path.dirname(path.dirname(claudeDbPath)));
  }

  ipcMain.handle('shell:open-external', async (_event, url: string) => {
    await shell.openExternal(url);
    return true;
  });

  // Now start services in the background (don't block the renderer)
  serviceManager.ensureAllServices()
    .then(() => {
      console.log('[Main] All services initialized');
      if (mainWindow) {
        serviceManager!.startStatusPolling(mainWindow);
      }
    })
    .catch(err => console.error('[Main] Service initialization error:', err));

}

// App lifecycle
app.whenReady().then(() => {
  createWindow();
  initializeServicesAndHandlers();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('before-quit', async () => {
  if (serviceManager) {
    await serviceManager.cleanup();
  }
});

// Export for testing
export { store, getDefaultClaudeStreamDbPath };

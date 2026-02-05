/**
 * MycelicMemory Desktop - Electron Main Process
 * Entry point for the Electron application
 */

import { app, BrowserWindow, ipcMain, shell } from 'electron';
import * as path from 'path';
import Store from 'electron-store';
import { registerMemoryHandlers } from './ipc/memory.ipc';
import { registerClaudeHandlers } from './ipc/claude.ipc';
import { registerExtractionHandlers } from './ipc/extraction.ipc';
import { registerConfigHandlers } from './ipc/config.ipc';
import { initSourcesIPC } from './ipc/sources.ipc';
import { ExtractionService } from './services/extraction-service';
import { MycelicMemoryClient } from './services/mycelicmemory-client';
import { AppSettings } from '../shared/types';

// Initialize electron-store for settings persistence
const store = new Store<{ settings: AppSettings }>({
  defaults: {
    settings: {
      api_url: 'http://localhost',
      api_port: 3099,
      ollama_base_url: 'http://localhost:11434',
      ollama_embedding_model: 'nomic-embed-text',
      ollama_chat_model: 'llama3.2',
      qdrant_url: 'http://localhost:6333',
      qdrant_enabled: true,
      claude_stream_db_path: getDefaultClaudeStreamDbPath(),
      extraction: {
        auto_extract: false,
        poll_interval_ms: 5000,
        min_message_length: 50,
        extract_tool_calls: true,
        extract_file_operations: true,
      },
      theme: 'dark',
      sidebar_collapsed: false,
    },
  },
});

let mainWindow: BrowserWindow | null = null;
let extractionService: ExtractionService | null = null;

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

function initializeServices(): void {
  const settings = store.get('settings');

  // Initialize extraction service
  extractionService = new ExtractionService({
    claudeDbPath: settings.claude_stream_db_path,
    apiUrl: `${settings.api_url}:${settings.api_port}`,
    config: settings.extraction,
    onProgress: (job) => {
      // Send progress updates to renderer
      if (mainWindow && !mainWindow.isDestroyed()) {
        mainWindow.webContents.send('extraction:progress', job);
      }
    },
  });

  // Start auto-extraction if enabled
  if (settings.extraction.auto_extract) {
    extractionService.start();
  }
}

function registerAllHandlers(): void {
  const settings = store.get('settings');
  const apiBaseUrl = `${settings.api_url}:${settings.api_port}`;
  const claudeDbPath = settings.claude_stream_db_path;

  // Create MycelicMemory client for sources IPC
  const client = new MycelicMemoryClient(apiBaseUrl);

  registerMemoryHandlers(ipcMain, apiBaseUrl);
  registerClaudeHandlers(ipcMain, claudeDbPath);
  registerExtractionHandlers(ipcMain, extractionService!);
  registerConfigHandlers(ipcMain, store);
  initSourcesIPC(client); // Register data source handlers

  // Handle external URL opening
  ipcMain.handle('shell:open-external', async (_event, url: string) => {
    await shell.openExternal(url);
    return true;
  });
}

// App lifecycle
app.whenReady().then(() => {
  createWindow();
  initializeServices();
  registerAllHandlers();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', () => {
  // Stop extraction service
  if (extractionService) {
    extractionService.stop();
  }

  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('before-quit', () => {
  if (extractionService) {
    extractionService.stop();
  }
});

// Export for testing
export { store, getDefaultClaudeStreamDbPath };

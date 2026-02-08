/**
 * Services IPC Handlers
 * Exposes service management to the renderer process
 */

import type { IpcMain } from 'electron';
import type { ServiceManager } from '../services/service-manager';

export function registerServicesHandlers(ipcMain: IpcMain, serviceManager: ServiceManager): void {
  // Get full status of all services
  ipcMain.handle('services:status', async () => {
    try {
      return await serviceManager.getFullStatus();
    } catch (error) {
      console.error('Failed to get service status:', error);
      return {
        backend: { running: false, managedByUs: false },
        ollama: { running: false, managedByUs: false },
        qdrant: { running: false, managedByUs: false },
      };
    }
  });

  // Start individual services
  ipcMain.handle('services:start-backend', async () => {
    try {
      return await serviceManager.startBackend();
    } catch (error) {
      console.error('Failed to start backend:', error);
      return false;
    }
  });

  ipcMain.handle('services:start-ollama', async () => {
    try {
      return await serviceManager.startOllama();
    } catch (error) {
      console.error('Failed to start Ollama:', error);
      return false;
    }
  });

  ipcMain.handle('services:start-qdrant', async () => {
    try {
      return await serviceManager.startQdrant();
    } catch (error) {
      console.error('Failed to start Qdrant:', error);
      return false;
    }
  });

  // Stop backend
  ipcMain.handle('services:stop-backend', async () => {
    try {
      await serviceManager.stopBackend();
      return true;
    } catch (error) {
      console.error('Failed to stop backend:', error);
      return false;
    }
  });
}

/**
 * Extraction IPC Handlers
 * Handles memory extraction operations
 */

import type { IpcMain } from 'electron';
import type { ExtractionService } from '../services/extraction-service';
import type { ExtractionConfig } from '../../shared/types';

export function registerExtractionHandlers(ipcMain: IpcMain, extractionService: ExtractionService): void {
  // Start extraction for a session
  ipcMain.handle('extraction:start', async (_event, sessionId: string) => {
    try {
      return await extractionService.extractSession(sessionId);
    } catch (error) {
      console.error('Failed to start extraction:', error);
      throw error;
    }
  });

  // Get extraction status
  ipcMain.handle('extraction:status', async () => {
    try {
      return extractionService.getStatus();
    } catch (error) {
      console.error('Failed to get extraction status:', error);
      throw error;
    }
  });

  // Get extraction config
  ipcMain.handle('extraction:config', async () => {
    try {
      return extractionService.getConfig();
    } catch (error) {
      console.error('Failed to get extraction config:', error);
      throw error;
    }
  });

  // Update extraction config
  ipcMain.handle('extraction:config:update', async (_event, config: ExtractionConfig) => {
    try {
      extractionService.updateConfig(config);
      return extractionService.getConfig();
    } catch (error) {
      console.error('Failed to update extraction config:', error);
      throw error;
    }
  });
}

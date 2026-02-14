/**
 * Config IPC Handlers
 * Handles application settings and configuration
 */

import type { IpcMain } from 'electron';
import type Store from 'electron-store';
import type { AppSettings } from '../../shared/types';
import { app } from 'electron';

export function registerConfigHandlers(
  ipcMain: IpcMain,
  store: Store<{ settings: AppSettings }>
): void {
  // Get all settings
  ipcMain.handle('settings:get', () => {
    try {
      return store.get('settings');
    } catch (error) {
      console.error('Failed to get settings:', error);
      throw error;
    }
  });

  // Update settings
  ipcMain.handle('settings:update', (_event, updates: Partial<AppSettings>) => {
    try {
      const current = store.get('settings');
      const updated = { ...current, ...updates };

      store.set('settings', updated);
      return updated;
    } catch (error) {
      console.error('Failed to update settings:', error);
      throw error;
    }
  });

  // Get app version
  ipcMain.handle('app:version', () => {
    return app.getVersion();
  });
}

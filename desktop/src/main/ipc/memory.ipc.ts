/**
 * Memory IPC Handlers
 * Handles memory-related IPC calls between renderer and main process
 */

import type { IpcMain } from 'electron';
import { MycelicMemoryClient } from '../services/mycelicmemory-client';
import type { MemoryCreateInput, MemoryUpdateInput, SearchOptions } from '../../shared/types';

export function registerMemoryHandlers(ipcMain: IpcMain, apiBaseUrl: string): void {
  const client = new MycelicMemoryClient(apiBaseUrl);

  // List memories
  ipcMain.handle('memory:list', async (_event, params?: { limit?: number; offset?: number; domain?: string }) => {
    try {
      return await client.getMemories(params);
    } catch (error) {
      console.error('Failed to list memories:', error);
      throw error;
    }
  });

  // Get single memory
  ipcMain.handle('memory:get', async (_event, id: string) => {
    try {
      return await client.getMemory(id);
    } catch (error) {
      console.error('Failed to get memory:', error);
      throw error;
    }
  });

  // Create memory
  ipcMain.handle('memory:create', async (_event, data: MemoryCreateInput) => {
    try {
      return await client.createMemory(data);
    } catch (error) {
      console.error('Failed to create memory:', error);
      throw error;
    }
  });

  // Update memory
  ipcMain.handle('memory:update', async (_event, id: string, data: MemoryUpdateInput) => {
    try {
      return await client.updateMemory(id, data);
    } catch (error) {
      console.error('Failed to update memory:', error);
      throw error;
    }
  });

  // Delete memory
  ipcMain.handle('memory:delete', async (_event, id: string) => {
    try {
      return await client.deleteMemory(id);
    } catch (error) {
      console.error('Failed to delete memory:', error);
      throw error;
    }
  });

  // Search memories
  ipcMain.handle('memory:search', async (_event, options: SearchOptions) => {
    try {
      return await client.searchMemories(options);
    } catch (error) {
      console.error('Failed to search memories:', error);
      throw error;
    }
  });

  // Get dashboard stats
  ipcMain.handle('stats:dashboard', async () => {
    try {
      return await client.getStats();
    } catch (error) {
      console.error('Failed to get stats:', error);
      throw error;
    }
  });

  // Health check
  ipcMain.handle('health:check', async () => {
    try {
      return await client.getHealth();
    } catch (error) {
      console.error('Health check failed:', error);
      return {
        api: false,
        ollama: false,
        qdrant: false,
        database: false,
      };
    }
  });

  // List domains
  ipcMain.handle('domains:list', async () => {
    try {
      return await client.getDomains();
    } catch (error) {
      console.error('Failed to list domains:', error);
      throw error;
    }
  });

  // Get relationships for a memory
  ipcMain.handle('relationships:get', async (_event, memoryId: string) => {
    try {
      return await client.getRelationships(memoryId);
    } catch (error) {
      console.error('Failed to get relationships:', error);
      throw error;
    }
  });

  // Discover relationships
  ipcMain.handle('relationships:discover', async () => {
    try {
      return await client.discoverRelationships();
    } catch (error) {
      console.error('Failed to discover relationships:', error);
      throw error;
    }
  });
}

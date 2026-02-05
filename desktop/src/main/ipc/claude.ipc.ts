/**
 * Claude IPC Handlers
 * Handles Claude Chat Stream database access
 */

import type { IpcMain } from 'electron';
import { ClaudeStreamDB } from '../services/claude-stream-db';

let db: ClaudeStreamDB | null = null;

export function registerClaudeHandlers(ipcMain: IpcMain, dbPath: string): void {
  // Initialize database connection lazily
  function getDB(): ClaudeStreamDB {
    if (!db) {
      db = new ClaudeStreamDB(dbPath);
    }
    return db;
  }

  // List projects
  ipcMain.handle('claude:projects', async () => {
    try {
      return getDB().getProjects();
    } catch (error) {
      console.error('Failed to list Claude projects:', error);
      throw error;
    }
  });

  // List sessions
  ipcMain.handle('claude:sessions', async (_event, projectId?: string) => {
    try {
      return getDB().getSessions(projectId);
    } catch (error) {
      console.error('Failed to list Claude sessions:', error);
      throw error;
    }
  });

  // Get single session
  ipcMain.handle('claude:session', async (_event, id: string) => {
    try {
      return getDB().getSession(id);
    } catch (error) {
      console.error('Failed to get Claude session:', error);
      throw error;
    }
  });

  // Get messages for a session
  ipcMain.handle('claude:messages', async (_event, sessionId: string) => {
    try {
      return getDB().getMessages(sessionId);
    } catch (error) {
      console.error('Failed to get Claude messages:', error);
      throw error;
    }
  });

  // Get tool calls for a session
  ipcMain.handle('claude:tool-calls', async (_event, sessionId: string) => {
    try {
      return getDB().getToolCalls(sessionId);
    } catch (error) {
      console.error('Failed to get Claude tool calls:', error);
      throw error;
    }
  });
}

// Cleanup function for app shutdown
export function closeClaudeDB(): void {
  if (db) {
    db.close();
    db = null;
  }
}

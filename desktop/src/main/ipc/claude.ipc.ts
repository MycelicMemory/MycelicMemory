/**
 * Claude IPC Handlers
 * Uses MycelicMemory backend REST API for chat history data
 */

import type { IpcMain } from 'electron';
import { MycelicMemoryClient } from '../services/mycelicmemory-client';

export function registerClaudeHandlers(ipcMain: IpcMain, client: MycelicMemoryClient): void {
  // List projects (unique project paths from ingested sessions)
  ipcMain.handle('claude:projects', async () => {
    try {
      const response = await client.get<any>('/chats/projects');
      const data = response.data || response;
      return Array.isArray(data) ? data : [];
    } catch (error) {
      console.error('Failed to list Claude projects:', error);
      return [];
    }
  });

  // List sessions (optionally filtered by project_path)
  ipcMain.handle('claude:sessions', async (_event, projectPath?: string) => {
    try {
      const params = new URLSearchParams();
      if (projectPath) params.set('project_path', projectPath);
      params.set('limit', '100');
      const query = params.toString();
      const response = await client.get<any>(`/chats${query ? `?${query}` : ''}`);
      const data = response.data || response;
      return Array.isArray(data) ? data : [];
    } catch (error) {
      console.error('Failed to list Claude sessions:', error);
      return [];
    }
  });

  // Get single session with messages
  ipcMain.handle('claude:session', async (_event, id: string) => {
    try {
      const response = await client.get<any>(`/chats/${id}`);
      const data = response.data || response;
      return data.session || data;
    } catch (error) {
      console.error('Failed to get Claude session:', error);
      return null;
    }
  });

  // Get messages for a session
  ipcMain.handle('claude:messages', async (_event, sessionId: string) => {
    try {
      const response = await client.get<any>(`/chats/${sessionId}/messages`);
      const data = response.data || response;
      return Array.isArray(data) ? data : [];
    } catch (error) {
      console.error('Failed to get Claude messages:', error);
      return [];
    }
  });

  // Get tool calls for a session
  ipcMain.handle('claude:tool-calls', async (_event, sessionId: string) => {
    try {
      const response = await client.get<any>(`/chats/${sessionId}/tool-calls`);
      const data = response.data || response;
      return Array.isArray(data) ? data : [];
    } catch (error) {
      console.error('Failed to get Claude tool calls:', error);
      return [];
    }
  });

  // Ingest conversations from Claude Code
  ipcMain.handle('claude:ingest', async (_event, opts?: { project_path?: string }) => {
    try {
      const response = await client.post<any>('/chats/ingest', {
        project_path: opts?.project_path || '',
        create_summaries: true,
        min_messages: 3,
      });
      return response.data || response;
    } catch (error) {
      console.error('Failed to ingest conversations:', error);
      throw error;
    }
  });

  // Search chat sessions
  ipcMain.handle('claude:search', async (_event, opts: { query: string; project_path?: string; limit?: number }) => {
    try {
      const params = new URLSearchParams();
      params.set('query', opts.query);
      if (opts.project_path) params.set('project_path', opts.project_path);
      if (opts.limit) params.set('limit', opts.limit.toString());
      const response = await client.get<any>(`/chats/search?${params.toString()}`);
      const data = response.data || response;
      return Array.isArray(data) ? data : [];
    } catch (error) {
      console.error('Failed to search chats:', error);
      return [];
    }
  });
}

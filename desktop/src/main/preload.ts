/**
 * MycelicMemory Desktop - Preload Script
 * Exposes safe IPC methods to the renderer process
 */

import { contextBridge, ipcRenderer } from 'electron';
import type {
  Memory,
  MemoryCreateInput,
  MemoryUpdateInput,
  SearchOptions,
  SearchResult,
  ClaudeProject,
  ClaudeSession,
  ClaudeMessage,
  ClaudeToolCall,
  ChatIngestResult,
  DashboardStats,
  HealthStatus,
  Domain,
  AppSettings,
  MemoryRelationship,
  DatabaseInfo,
  ClaudeChatStreamStatus,
  ServiceStatus,
} from '../shared/types';

// Type-safe IPC invoke wrapper
function invoke<T>(channel: string, ...args: unknown[]): Promise<T> {
  return ipcRenderer.invoke(channel, ...args);
}

// API exposed to renderer
const api = {
  // Memory operations
  memory: {
    list: (params?: { limit?: number; offset?: number; domain?: string }): Promise<Memory[]> =>
      invoke('memory:list', params),
    get: (id: string): Promise<Memory | null> =>
      invoke('memory:get', id),
    create: (data: MemoryCreateInput): Promise<Memory> =>
      invoke('memory:create', data),
    store: (data: MemoryCreateInput): Promise<Memory> =>
      invoke('memory:create', data),
    update: (id: string, data: MemoryUpdateInput): Promise<Memory> =>
      invoke('memory:update', id, data),
    delete: (id: string): Promise<boolean> =>
      invoke('memory:delete', id),
    search: (options: SearchOptions): Promise<SearchResult[]> =>
      invoke('memory:search', options),
  },

  // Claude Code Chat History operations (via MycelicMemory backend)
  claude: {
    projects: (): Promise<ClaudeProject[]> =>
      invoke('claude:projects'),
    sessions: (projectPath?: string): Promise<ClaudeSession[]> =>
      invoke('claude:sessions', projectPath),
    session: (id: string): Promise<ClaudeSession | null> =>
      invoke('claude:session', id),
    messages: (sessionId: string): Promise<ClaudeMessage[]> =>
      invoke('claude:messages', sessionId),
    toolCalls: (sessionId: string): Promise<ClaudeToolCall[]> =>
      invoke('claude:tool-calls', sessionId),
    ingest: (projectPath?: string): Promise<ChatIngestResult> =>
      invoke('claude:ingest', { project_path: projectPath }),
    search: (query: string, projectPath?: string, limit?: number): Promise<ClaudeSession[]> =>
      invoke('claude:search', { query, project_path: projectPath, limit }),
  },

  // Stats & Health
  stats: {
    dashboard: (): Promise<DashboardStats> =>
      invoke('stats:dashboard'),
    health: (): Promise<HealthStatus> =>
      invoke('health:check'),
  },

  // Domains
  domains: {
    list: (): Promise<Domain[]> =>
      invoke('domains:list'),
  },

  // Relationships (for knowledge graph)
  relationships: {
    get: (memoryId: string): Promise<MemoryRelationship[]> =>
      invoke('relationships:get', memoryId),
    getAll: (params?: { limit?: number; min_strength?: number }): Promise<MemoryRelationship[]> =>
      invoke('relationships:getAll', params),
    discover: (opts?: { method?: string }): Promise<MemoryRelationship[]> =>
      invoke('relationships:discover', opts),
  },

  // Database management
  databases: {
    list: (): Promise<DatabaseInfo[]> =>
      invoke('databases:list'),
    create: (data: { name: string; description?: string }): Promise<DatabaseInfo> =>
      invoke('databases:create', data),
    get: (name: string): Promise<DatabaseInfo> =>
      invoke('databases:get', name),
    delete: (name: string): Promise<boolean> =>
      invoke('databases:delete', name),
    switch: (name: string): Promise<boolean> =>
      invoke('databases:switch', name),
    archive: (name: string): Promise<{ backup_path: string }> =>
      invoke('databases:archive', name),
    export: (name: string): Promise<{ path: string } | null> =>
      invoke('databases:export', name),
    import: (): Promise<{ path: string } | null> =>
      invoke('databases:import'),
    importConfirm: (data: { path: string; name: string }): Promise<DatabaseInfo> =>
      invoke('databases:import-confirm', data),
  },

  // Settings
  settings: {
    get: (): Promise<AppSettings> =>
      invoke('settings:get'),
    update: (settings: Partial<AppSettings>): Promise<AppSettings> =>
      invoke('settings:update', settings),
  },

  // Claude Chat Stream daemon control
  claudeStream: {
    status: (): Promise<ClaudeChatStreamStatus> =>
      invoke('claude-stream:status'),
    isRunning: (): Promise<boolean> =>
      invoke('claude-stream:is-running'),
    start: (): Promise<boolean> =>
      invoke('claude-stream:start'),
    stop: (): Promise<boolean> =>
      invoke('claude-stream:stop'),
    connectSSE: (): Promise<boolean> =>
      invoke('claude-stream:connect-sse'),
    disconnectSSE: (): Promise<boolean> =>
      invoke('claude-stream:disconnect-sse'),
    getLogs: (limit?: number, level?: string): Promise<unknown[]> =>
      invoke('claude-stream:logs', limit, level),
    getStats: (): Promise<unknown> =>
      invoke('claude-stream:stats'),
    // Event listeners for SSE events
    onEvent: (eventType: string, callback: (data: unknown) => void): (() => void) => {
      const handler = (_event: Electron.IpcRendererEvent, data: unknown) => callback(data);
      ipcRenderer.on(`claude-stream:event:${eventType}`, handler);
      return () => ipcRenderer.removeListener(`claude-stream:event:${eventType}`, handler);
    },
  },

  // Service management
  services: {
    status: (): Promise<ServiceStatus> =>
      invoke('services:status'),
    startBackend: (): Promise<boolean> =>
      invoke('services:start-backend'),
    startOllama: (): Promise<boolean> =>
      invoke('services:start-ollama'),
    startQdrant: (): Promise<boolean> =>
      invoke('services:start-qdrant'),
    stopBackend: (): Promise<boolean> =>
      invoke('services:stop-backend'),
    onStatusUpdate: (callback: (status: ServiceStatus) => void): (() => void) => {
      const handler = (_event: Electron.IpcRendererEvent, status: ServiceStatus) => callback(status);
      ipcRenderer.on('services:status-update', handler);
      return () => ipcRenderer.removeListener('services:status-update', handler);
    },
  },

  // Shell operations
  shell: {
    openExternal: (url: string): Promise<boolean> =>
      invoke('shell:open-external', url),
  },

  // App info
  app: {
    getVersion: (): Promise<string> =>
      invoke('app:version'),
    getPlatform: (): string =>
      process.platform,
  },
};

// Expose the API to the renderer
contextBridge.exposeInMainWorld('mycelicMemory', api);

// Type declaration for renderer
declare global {
  interface Window {
    mycelicMemory: typeof api;
  }
}

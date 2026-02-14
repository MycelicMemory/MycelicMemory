/**
 * API Bridge - Provides a unified API interface that works in both Electron and browser environments
 * In Electron: Uses window.mycelicMemory (IPC)
 * In Browser: Falls back to direct fetch calls for testing
 */

const API_BASE = 'http://127.0.0.1:3099/api/v1';

// Check if we're in Electron
const isElectron = typeof window !== 'undefined' && window.mycelicMemory !== undefined;

// Direct fetch wrapper for browser testing
async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!response.ok) {
    throw new Error(`API Error: ${response.status}`);
  }

  const data = await response.json();
  return data.data ?? data;
}

// Browser fallback implementations
const browserApi = {
  memory: {
    list: async (params?: { limit?: number; offset?: number; domain?: string }) => {
      const searchParams = new URLSearchParams();
      if (params?.limit) searchParams.set('limit', params.limit.toString());
      if (params?.offset) searchParams.set('offset', params.offset.toString());
      if (params?.domain) searchParams.set('domain', params.domain);
      const query = searchParams.toString();
      const result = await fetchApi<any[]>(`/memories${query ? `?${query}` : ''}`);
      // API returns [{memory: {...}}, ...] - extract the memory objects
      if (Array.isArray(result) && result[0]?.memory) {
        return result.map((item: any) => item.memory);
      }
      return result;
    },
    get: async (id: string) => fetchApi<any>(`/memories/${id}`),
    create: async (data: any) => fetchApi<any>('/memories', { method: 'POST', body: JSON.stringify(data) }),
    update: async (id: string, data: any) => fetchApi<any>(`/memories/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: async (id: string) => fetchApi<void>(`/memories/${id}`, { method: 'DELETE' }),
    search: async (options: any) => fetchApi<any[]>('/memories/search', { method: 'POST', body: JSON.stringify(options) }),
  },
  stats: {
    dashboard: async () => {
      const result = await fetchApi<any>('/stats');
      return {
        memory_count: result.total_memories ?? result.memory_count ?? 0,
        session_count: result.session_count ?? 0,
        domain_count: result.domain_count ?? (result.unique_tags?.length ?? 0),
        this_week_count: 0,
      };
    },
    health: async () => {
      try {
        const result = await fetchApi<any>('/health');
        return {
          api: true,
          ollama: result.ollama ?? false,
          qdrant: result.qdrant ?? false,
          database: result.database ?? true,
        };
      } catch {
        return { api: false, ollama: false, qdrant: false, database: false };
      }
    },
  },
  domains: {
    list: async () => {
      const result = await fetchApi<any>('/domains');
      return result.domains ?? result ?? [];
    },
  },
  relationships: {
    get: async (memoryId: string) => fetchApi<any[]>(`/relationships?memory_id=${memoryId}`),
    discover: async () => fetchApi<any[]>('/relationships/discover', { method: 'POST' }),
  },
  claude: {
    projects: async () => {
      const result = await fetchApi<any[]>('/chats/projects');
      return Array.isArray(result) ? result : [];
    },
    sessions: async (projectPath?: string) => {
      const params = new URLSearchParams();
      if (projectPath) params.set('project_path', projectPath);
      params.set('limit', '100');
      const query = params.toString();
      const result = await fetchApi<any[]>(`/chats${query ? `?${query}` : ''}`);
      return Array.isArray(result) ? result : [];
    },
    session: async (id: string) => {
      const result = await fetchApi<any>(`/chats/${id}`);
      return result?.session || result;
    },
    messages: async (sessionId: string) => {
      const result = await fetchApi<any[]>(`/chats/${sessionId}/messages`);
      return Array.isArray(result) ? result : [];
    },
    toolCalls: async (sessionId: string) => {
      const result = await fetchApi<any[]>(`/chats/${sessionId}/tool-calls`);
      return Array.isArray(result) ? result : [];
    },
    ingest: async (projectPath?: string) => {
      return fetchApi<any>('/chats/ingest', {
        method: 'POST',
        body: JSON.stringify({ project_path: projectPath || '', create_summaries: true, min_messages: 3 }),
      });
    },
    search: async (query: string, projectPath?: string, limit?: number) => {
      const params = new URLSearchParams();
      params.set('query', query);
      if (projectPath) params.set('project_path', projectPath);
      if (limit) params.set('limit', limit.toString());
      const result = await fetchApi<any[]>(`/chats/search?${params.toString()}`);
      return Array.isArray(result) ? result : [];
    },
  },
  config: {
    get: async () => ({}),
    set: async () => {},
    getAll: async () => ({}),
  },
  services: {
    status: async () => {
      try {
        const health = await browserApi.stats.health();
        return {
          backend: { running: health.api, managedByUs: false },
          ollama: { running: health.ollama, managedByUs: false },
          qdrant: { running: health.qdrant, managedByUs: false },
        };
      } catch {
        return {
          backend: { running: false, managedByUs: false },
          ollama: { running: false, managedByUs: false },
          qdrant: { running: false, managedByUs: false },
        };
      }
    },
    startBackend: async () => false,
    startOllama: async () => false,
    startQdrant: async () => false,
    stopBackend: async () => false,
    onStatusUpdate: () => () => {},
  },
};

// Export the appropriate API based on environment
export const api = isElectron ? window.mycelicMemory : browserApi;

// Also set on window for components that access window.mycelicMemory directly
if (!isElectron && typeof window !== 'undefined') {
  (window as any).mycelicMemory = browserApi;
}

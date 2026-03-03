/**
 * API Bridge - Provides a unified API interface that works in both Electron and browser environments
 * In Electron: Uses window.mycelicMemory (IPC)
 * In Browser: Falls back to direct fetch calls for testing
 */

const API_BASE = 'http://127.0.0.1:3002/api/v1';

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
    store: async (data: any) => fetchApi<any>('/memories', { method: 'POST', body: JSON.stringify(data) }),
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
    create: async (data: { name: string; description?: string }) =>
      fetchApi<any>('/domains', { method: 'POST', body: JSON.stringify(data) }),
    stats: async (domain: string) => fetchApi<any>(`/domains/${encodeURIComponent(domain)}/stats`),
  },
  categories: {
    list: async () => fetchApi<any[]>('/categories'),
    create: async (data: { name: string; description?: string; parent_id?: string }) =>
      fetchApi<any>('/categories', { method: 'POST', body: JSON.stringify(data) }),
    stats: async () => fetchApi<any>('/categories/stats'),
    categorize: async (memoryId: string, data: { category_id: string; confidence?: number }) =>
      fetchApi<any>(`/memories/${memoryId}/categorize`, { method: 'POST', body: JSON.stringify(data) }),
  },
  relationships: {
    getAll: async (params?: { limit?: number; min_strength?: number }) => {
      const searchParams = new URLSearchParams();
      if (params?.limit) searchParams.set('limit', params.limit.toString());
      if (params?.min_strength) searchParams.set('min_strength', params.min_strength.toString());
      const query = searchParams.toString();
      return fetchApi<any[]>(`/relationships${query ? `?${query}` : ''}`);
    },
    get: async (memoryId: string) => fetchApi<any[]>(`/relationships?memory_id=${memoryId}`),
    create: async (data: { source_memory_id: string; target_memory_id: string; relationship_type_enum: string; strength?: number; context?: string }) =>
      fetchApi<any>('/relationships', { method: 'POST', body: JSON.stringify(data) }),
    discover: async (opts?: { method?: string }) => fetchApi<any[]>('/relationships/discover', { method: 'POST', body: JSON.stringify(opts ?? {}) }),
    batchDiscover: async (data?: { limit?: number; min_score?: number; domain?: string }) =>
      fetchApi<any>('/relationships/batch-discover', { method: 'POST', body: JSON.stringify(data ?? {}) }),
    related: async (memoryId: string, limit?: number) => {
      const params = new URLSearchParams();
      if (limit) params.set('limit', limit.toString());
      return fetchApi<any[]>(`/memories/${memoryId}/related?${params.toString()}`);
    },
    graph: async (memoryId: string, depth?: number) => {
      const params = new URLSearchParams();
      if (depth) params.set('depth', depth.toString());
      return fetchApi<any>(`/memories/${memoryId}/graph?${params.toString()}`);
    },
  },
  graph: {
    stats: async () => fetchApi<any>('/graph/stats'),
  },
  analysis: {
    analyze: async (data: { analysis_type: string; question?: string; query?: string; timeframe?: string; limit?: number; domain?: string }) =>
      fetchApi<any>('/analyze', { method: 'POST', body: JSON.stringify(data) }),
  },
  sources: {
    list: async (params?: { source_type?: string; status?: string }) => {
      const searchParams = new URLSearchParams();
      if (params?.source_type) searchParams.set('source_type', params.source_type);
      if (params?.status) searchParams.set('status', params.status);
      const query = searchParams.toString();
      return fetchApi<any[]>(`/sources${query ? `?${query}` : ''}`);
    },
    get: async (id: string) => fetchApi<any>(`/sources/${id}`),
    create: async (data: { source_type: string; name: string; config?: string }) =>
      fetchApi<any>('/sources', { method: 'POST', body: JSON.stringify(data) }),
    update: async (id: string, data: { name?: string; config?: string; status?: string }) =>
      fetchApi<any>(`/sources/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: async (id: string) => fetchApi<void>(`/sources/${id}`, { method: 'DELETE' }),
    pause: async (id: string) => fetchApi<any>(`/sources/${id}/pause`, { method: 'POST' }),
    resume: async (id: string) => fetchApi<any>(`/sources/${id}/resume`, { method: 'POST' }),
    sync: async (id: string) => fetchApi<any>(`/sources/${id}/sync`, { method: 'POST' }),
    history: async (id: string, limit?: number) => {
      const params = new URLSearchParams();
      if (limit) params.set('limit', limit.toString());
      return fetchApi<any[]>(`/sources/${id}/history?${params.toString()}`);
    },
    stats: async (id: string) => fetchApi<any>(`/sources/${id}/stats`),
    memories: async (id: string, params?: { limit?: number; offset?: number }) => {
      const searchParams = new URLSearchParams();
      if (params?.limit) searchParams.set('limit', params.limit.toString());
      if (params?.offset) searchParams.set('offset', params.offset.toString());
      return fetchApi<any[]>(`/sources/${id}/memories?${searchParams.toString()}`);
    },
  },
  trace: {
    source: async (memoryId: string) => fetchApi<any>(`/memories/${memoryId}/trace`),
  },
  recall: {
    query: async (data: { context: string; files?: string[]; project?: string; limit?: number; depth?: number }) =>
      fetchApi<any>('/recall', { method: 'POST', body: JSON.stringify(data) }),
  },
  search: {
    tags: async (data: { tags: string[]; tag_operator?: string; limit?: number; domain?: string }) =>
      fetchApi<any[]>('/search/tags', { method: 'POST', body: JSON.stringify(data) }),
    dateRange: async (data: { start_date: string; end_date: string; limit?: number; domain?: string }) =>
      fetchApi<any[]>('/search/date-range', { method: 'POST', body: JSON.stringify(data) }),
    intelligent: async (data: any) =>
      fetchApi<any[]>('/memories/search/intelligent', { method: 'POST', body: JSON.stringify(data) }),
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
  databases: {
    list: async () => fetchApi<any[]>('/databases'),
    create: async (data: { name: string; description?: string }) =>
      fetchApi<any>('/databases', { method: 'POST', body: JSON.stringify(data) }),
    get: async (name: string) => fetchApi<any>(`/databases/${encodeURIComponent(name)}`),
    delete: async (name: string) => fetchApi<void>(`/databases/${encodeURIComponent(name)}`, { method: 'DELETE' }),
    switch: async (name: string) => fetchApi<any>(`/databases/${encodeURIComponent(name)}/switch`, { method: 'POST' }),
    archive: async (name: string) => fetchApi<any>(`/databases/${encodeURIComponent(name)}/archive`, { method: 'POST' }),
  },
  models: {
    list: async () => fetchApi<any>('/models'),
    pull: async (name: string) => fetchApi<any>('/models/pull', { method: 'POST', body: JSON.stringify({ name }) }),
    test: async () => fetchApi<any>('/models/test', { method: 'POST' }),
    status: async () => fetchApi<any>('/models/status'),
  },
  config: {
    get: async () => ({}),
    set: async () => {},
    getAll: async () => ({}),
    updateOllama: async (data: { base_url?: string; embedding_model?: string; chat_model?: string }) =>
      fetchApi<any>('/config/ollama', { method: 'PUT', body: JSON.stringify(data) }),
    updateQdrant: async (data: { url?: string; api_key?: string }) =>
      fetchApi<any>('/config/qdrant', { method: 'PUT', body: JSON.stringify(data) }),
  },
  reindex: {
    start: async () => fetchApi<any>('/memories/reindex', { method: 'POST' }),
  },
  seed: {
    populate: async () => fetchApi<any>('/seed', { method: 'POST' }),
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

// ============================================
// MycelicMemory Desktop - Shared Types
// ============================================

// Memory types (from MycelicMemory API)
export interface Memory {
  id: string;
  content: string;
  domain?: string;
  source?: string;
  importance: number;
  tags?: string[];
  created_at: string;
  updated_at: string;
  session_id?: string;
}

export interface MemoryCreateInput {
  content: string;
  domain?: string;
  source?: string;
  importance?: number;
  tags?: string[];
}

export interface MemoryUpdateInput {
  content?: string;
  importance?: number;
  tags?: string[];
}

export interface SearchResult {
  memory: Memory;
  score?: number;
  similarity?: number;
}

export interface SearchOptions {
  query: string;
  search_type?: 'semantic' | 'keyword' | 'hybrid' | 'tags';
  domain?: string;
  tags?: string[];
  limit?: number;
  use_ai?: boolean;
}

// Domain types
export interface Domain {
  id?: string;
  name: string;
  description?: string;
  memory_count?: number;
}

// Session types (from MycelicMemory)
export interface MemorySession {
  id: string;
  started_at: string;
  memory_count: number;
}

// Claude Chat Stream types
export interface ClaudeProject {
  id: string;
  original_path: string;
  display_name?: string;
  discovered_at: string;
  last_activity?: string;
  session_count: number;
  message_count: number;
}

export interface ClaudeSession {
  id: string;
  project_id: string;
  file_path: string;
  first_prompt?: string;
  summary?: string;
  git_branch?: string;
  is_sidechain: boolean;
  is_subagent: boolean;
  parent_session_id?: string;
  message_count: number;
  user_message_count: number;
  assistant_message_count: number;
  tool_call_count: number;
  started_at: string;
  ended_at?: string;
}

export interface ClaudeMessage {
  id: string;
  session_id: string;
  parent_uuid?: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  message_type?: string;
  timestamp: string;
  cwd?: string;
  git_branch?: string;
  thinking_tokens?: number;
  has_tool_calls: boolean;
}

export interface ClaudeToolCall {
  id: string;
  message_id: string;
  session_id: string;
  tool_name: string;
  tool_id?: string;
  parameters?: string;
  result?: string;
  result_truncated: boolean;
  success?: boolean;
  error_message?: string;
  duration_ms?: number;
  sequence_number?: number;
  timestamp: string;
}

export interface ClaudeFileReference {
  id: number;
  message_id?: string;
  session_id: string;
  tool_call_id?: string;
  file_path: string;
  operation: 'read' | 'write' | 'edit' | 'delete';
  old_content_preview?: string;
  new_content_preview?: string;
  lines_changed?: number;
  timestamp: string;
}

export interface ChangeStreamEntry {
  id: number;
  entity_type: 'session' | 'message';
  entity_id: string;
  change_type: 'insert' | 'update' | 'delete';
  payload: string;
  created_at: string;
  processed_by?: string;
}

// Extraction types
export interface ExtractionJob {
  id: string;
  session_id: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  messages_processed: number;
  memories_created: number;
  started_at?: string;
  completed_at?: string;
  error?: string;
}

export interface ExtractionConfig {
  auto_extract: boolean;
  poll_interval_ms: number;
  min_message_length: number;
  extract_tool_calls: boolean;
  extract_file_operations: boolean;
}

// Relationship types (for knowledge graph)
export interface MemoryRelationship {
  id: string;
  source_id: string;
  target_id: string;
  relationship_type: 'references' | 'contradicts' | 'expands' | 'similar' | 'sequential' | 'causes' | 'enables';
  strength: number;
  created_at: string;
}

// Settings types
export interface AppSettings {
  // MycelicMemory API
  api_url: string;
  api_port: number;

  // Ollama
  ollama_base_url: string;
  ollama_embedding_model: string;
  ollama_chat_model: string;

  // Qdrant
  qdrant_url: string;
  qdrant_enabled: boolean;

  // Claude Chat Stream
  claude_stream_db_path: string;

  // Extraction
  extraction: ExtractionConfig;

  // UI
  theme: 'dark' | 'light' | 'system';
  sidebar_collapsed: boolean;
}

// Stats types
export interface DashboardStats {
  memory_count: number;
  session_count: number;
  domain_count: number;
  this_week_count: number;
  last_extraction?: string;
}

export interface HealthStatus {
  api: boolean;
  ollama: boolean;
  qdrant: boolean;
  database: boolean;
}

// =============================================================================
// DATA SOURCE TYPES (Multi-source ingestion)
// =============================================================================

export type DataSourceType = 'claude-stream' | 'slack' | 'email' | 'browser' | 'notion' | 'obsidian' | 'github' | 'custom';
export type DataSourceStatus = 'active' | 'paused' | 'error';
export type SyncStatus = 'running' | 'completed' | 'failed';

export interface DataSource {
  id: string;
  source_type: DataSourceType;
  name: string;
  config: string; // JSON
  status: DataSourceStatus;
  last_sync_at?: string;
  last_sync_position?: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export interface DataSourceCreateInput {
  source_type: DataSourceType;
  name: string;
  config?: string;
}

export interface DataSourceUpdateInput {
  name?: string;
  config?: string;
  status?: DataSourceStatus;
}

export interface SyncHistoryEntry {
  id: string;
  source_id: string;
  started_at: string;
  completed_at?: string;
  items_processed: number;
  memories_created: number;
  duplicates_skipped: number;
  status: SyncStatus;
  error?: string;
}

export interface DataSourceStats {
  total_memories: number;
  total_syncs: number;
  successful_syncs: number;
  failed_syncs: number;
  last_sync_at?: string;
  last_error?: string;
}

export interface IngestItem {
  external_id: string;
  content: string;
  content_type?: 'text' | 'code' | 'markdown' | 'html';
  timestamp?: string;
  metadata?: {
    author?: string;
    channel?: string;
    thread_id?: string;
    domain?: string;
    importance?: number;
    tags?: string[];
  };
}

export interface IngestRequest {
  items: IngestItem[];
  checkpoint?: string;
}

export interface IngestResponse {
  processed: number;
  memories_created: number;
  duplicates_skipped: number;
  checkpoint: string;
}

// Slack-specific configuration
export interface SlackSourceConfig {
  bot_token?: string;
  user_token?: string;
  channels: {
    mode: 'include' | 'exclude' | 'all';
    list: string[];
  };
  dms: {
    enabled: boolean;
    mode: 'include' | 'exclude' | 'all';
    list: string[];
  };
  sync: {
    interval_minutes: number;
    backfill_days?: number;
  };
  filters: {
    min_message_length: number;
    exclude_bot_messages: boolean;
    include_threads: boolean;
  };
}

// Claude Stream-specific configuration
export interface ClaudeStreamSourceConfig {
  db_path: string;
  auto_extract: boolean;
  poll_interval_ms: number;
  extract_tool_calls: boolean;
  extract_file_operations: boolean;
}

// IPC Channel types
export type IPCChannels = {
  // Memory operations
  'memory:list': { params: { limit?: number; offset?: number; domain?: string }; result: Memory[] };
  'memory:get': { params: { id: string }; result: Memory | null };
  'memory:create': { params: MemoryCreateInput; result: Memory };
  'memory:update': { params: { id: string; data: MemoryUpdateInput }; result: Memory };
  'memory:delete': { params: { id: string }; result: boolean };
  'memory:search': { params: SearchOptions; result: SearchResult[] };

  // Claude sessions
  'claude:projects': { params: void; result: ClaudeProject[] };
  'claude:sessions': { params: { project_id?: string }; result: ClaudeSession[] };
  'claude:session': { params: { id: string }; result: ClaudeSession | null };
  'claude:messages': { params: { session_id: string }; result: ClaudeMessage[] };
  'claude:tool-calls': { params: { session_id: string }; result: ClaudeToolCall[] };

  // Extraction
  'extraction:start': { params: { session_id: string }; result: ExtractionJob };
  'extraction:status': { params: void; result: ExtractionJob[] };
  'extraction:config': { params: void; result: ExtractionConfig };
  'extraction:config:update': { params: ExtractionConfig; result: ExtractionConfig };

  // Stats & Health
  'stats:dashboard': { params: void; result: DashboardStats };
  'health:check': { params: void; result: HealthStatus };

  // Domains
  'domains:list': { params: void; result: Domain[] };

  // Settings
  'settings:get': { params: void; result: AppSettings };
  'settings:update': { params: Partial<AppSettings>; result: AppSettings };

  // Relationships (for graph)
  'relationships:get': { params: { memory_id: string }; result: MemoryRelationship[] };
  'relationships:discover': { params: void; result: MemoryRelationship[] };

  // Data Sources
  'sources:list': { params: { source_type?: DataSourceType; status?: DataSourceStatus }; result: DataSource[] };
  'sources:get': { params: { id: string }; result: DataSource | null };
  'sources:create': { params: DataSourceCreateInput; result: DataSource };
  'sources:update': { params: { id: string; data: DataSourceUpdateInput }; result: DataSource };
  'sources:delete': { params: { id: string }; result: boolean };
  'sources:pause': { params: { id: string }; result: DataSource };
  'sources:resume': { params: { id: string }; result: DataSource };
  'sources:sync': { params: { id: string }; result: SyncHistoryEntry };
  'sources:ingest': { params: { id: string; request: IngestRequest }; result: IngestResponse };
  'sources:history': { params: { id: string; limit?: number }; result: SyncHistoryEntry[] };
  'sources:stats': { params: { id: string }; result: DataSourceStats };
  'sources:memories': { params: { id: string; limit?: number; offset?: number }; result: Memory[] };
};

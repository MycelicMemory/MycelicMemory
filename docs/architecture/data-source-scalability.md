# MycelicMemory Data Source Scalability Architecture

## Overview

This document outlines the architecture for scaling MycelicMemory to support multiple continuous data streams beyond claude-chat-stream, including Slack, email, browser history, and other sources.

## Current Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     MycelicMemory Core                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  │
│  │   Memory    │  │   Qdrant    │  │   Ollama    │                  │
│  │   SQLite    │  │   Vector    │  │   Embedding │                  │
│  │   + FTS5    │  │   Store     │  │   + Chat    │                  │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                  │
│         └────────────────┴─────────────────┘                         │
│                          │                                           │
│                   REST API (:3099)                                   │
└──────────────────────────┬───────────────────────────────────────────┘
                           │
┌──────────────────────────┴───────────────────────────────────────────┐
│                  Desktop App (Current)                               │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                   ExtractionService                              ││
│  │  - Polls claude-chat-stream change_stream                        ││
│  │  - Extracts memories via API                                     ││
│  └─────────────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────────────┘
```

## Scalable Architecture for Multiple Data Sources

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          MycelicMemory Core                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Memory    │  │   Qdrant    │  │   Ollama    │  │  Source     │        │
│  │   SQLite    │  │   Vector    │  │   Embedding │  │  Registry   │        │
│  │   + FTS5    │  │   Store     │  │   + Chat    │  │  (new)      │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
│         └────────────────┴─────────────────┴────────────────┘               │
│                                    │                                         │
│                   REST API (:3099) + Source Management                       │
│                   POST /api/v1/sources/register                              │
│                   POST /api/v1/sources/{id}/ingest                           │
│                   GET  /api/v1/sources/{id}/status                           │
└────────────────────────────────────┬─────────────────────────────────────────┘
                                     │
        ┌────────────────────────────┼────────────────────────────────┐
        │                            │                                │
        ▼                            ▼                                ▼
┌───────────────────┐  ┌────────────────────────┐  ┌───────────────────────┐
│ Claude Stream     │  │    Slack Exporter      │  │   Future Sources      │
│ Adapter           │  │    (New Service)       │  │   - Email             │
│ ─────────────────│  │ ──────────────────────│  │   - Browser History   │
│ - Reads chats.db │  │ - Slack Web API        │  │   - Notion            │
│ - Monitors       │  │ - OAuth2 + Bot Token   │  │   - Obsidian          │
│   change_stream  │  │ - Configurable         │  │   - GitHub            │
│ - Transforms to  │  │   channels/DMs         │  │   - Jira              │
│   memory format  │  │ - Backfill support     │  │   - etc.              │
└───────────────────┘  │ - Incremental sync    │  └───────────────────────┘
                       │ - Rate limiting       │
                       └────────────────────────┘
```

## Core Components

### 1. Source Registry (New in MycelicMemory Core)

```typescript
// New table: data_sources
interface DataSource {
  id: string;                    // UUID
  source_type: string;           // 'claude-stream' | 'slack' | 'email' | ...
  name: string;                  // Display name
  config: SourceConfig;          // Source-specific configuration
  status: 'active' | 'paused' | 'error';
  last_sync_at: string;
  last_sync_position: string;    // Cursor/checkpoint for incremental sync
  error_message?: string;
  created_at: string;
  updated_at: string;
}

// New table: data_source_sync_history
interface SyncHistoryEntry {
  id: string;
  source_id: string;
  started_at: string;
  completed_at?: string;
  items_processed: number;
  memories_created: number;
  status: 'running' | 'completed' | 'failed';
  error?: string;
}
```

### 2. Unified Ingestion API

```typescript
// POST /api/v1/sources/{source_id}/ingest
interface IngestRequest {
  items: IngestItem[];
  checkpoint?: string;  // For resumable sync
}

interface IngestItem {
  external_id: string;          // Unique ID in source system
  content: string;              // Raw content
  content_type: 'text' | 'code' | 'markdown' | 'html';
  metadata: {
    source_type: string;        // 'slack' | 'claude-stream' | etc.
    timestamp: string;
    author?: string;
    channel?: string;           // For Slack
    thread_id?: string;
    file_references?: string[];
    [key: string]: any;         // Source-specific metadata
  };
}

interface IngestResponse {
  processed: number;
  memories_created: number;
  duplicates_skipped: number;
  checkpoint: string;
}
```

### 3. Content Processor Pipeline

```
IngestItem → ContentProcessor → Memory

ContentProcessor stages:
1. Deduplication (hash-based)
2. Content normalization
3. Importance scoring
4. Domain classification
5. Tag extraction
6. Embedding generation
7. Memory storage
```

## Slack Exporter Service Architecture

### Directory Structure

```
mycelicmemory-slack-exporter/
├── package.json
├── tsconfig.json
├── .env.example
├── src/
│   ├── index.ts                 # Entry point
│   ├── config/
│   │   └── settings.ts          # Configuration schema
│   ├── slack/
│   │   ├── client.ts            # Slack Web API wrapper
│   │   ├── auth.ts              # OAuth2 flow
│   │   └── types.ts             # Slack API types
│   ├── sync/
│   │   ├── scheduler.ts         # Cron-based scheduling
│   │   ├── incremental.ts       # Incremental sync logic
│   │   ├── backfill.ts          # Historical data fetch
│   │   └── checkpoint.ts        # Sync state persistence
│   ├── transform/
│   │   ├── message.ts           # Message → IngestItem
│   │   ├── thread.ts            # Thread aggregation
│   │   └── attachment.ts        # File/attachment handling
│   └── api/
│       ├── server.ts            # Config API (optional)
│       └── routes.ts
└── data/
    └── sync-state.json          # Local checkpoint storage
```

### Configuration Schema

```typescript
interface SlackExporterConfig {
  // Slack credentials
  slack: {
    bot_token: string;           // xoxb-...
    user_token?: string;         // xoxp-... (for DMs with OAuth)
    app_token?: string;          // xapp-... (for Socket Mode)
  };

  // MycelicMemory connection
  mycelicmemory: {
    api_url: string;
    source_id: string;           // Registered source ID
  };

  // Sync configuration
  sync: {
    interval_minutes: number;    // Default: 60
    batch_size: number;          // Messages per API call
    rate_limit_delay_ms: number; // Between API calls
    backfill_days?: number;      // How far back to fetch initially
  };

  // Channel/DM selection
  channels: {
    mode: 'include' | 'exclude' | 'all';
    list: string[];              // Channel IDs or names
  };

  dms: {
    enabled: boolean;
    mode: 'include' | 'exclude' | 'all';
    list: string[];              // User IDs for specific DMs
  };

  // Content filtering
  filters: {
    min_message_length: number;
    exclude_bot_messages: boolean;
    exclude_system_messages: boolean;
    include_threads: boolean;
    include_reactions: boolean;
    include_attachments: boolean;
  };
}
```

### Sync Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Scheduler (cron)                              │
│                    runs every N minutes                              │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Load Sync State                                 │
│  - Last sync timestamp per channel                                   │
│  - Last cursor/oldest message ID                                     │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    For each enabled channel:                         │
│  1. conversations.history (since last_sync_ts)                       │
│  2. For threads: conversations.replies                               │
│  3. Rate limit handling (429 → backoff)                              │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Transform Messages                              │
│  - Clean markdown/formatting                                         │
│  - Resolve user mentions → names                                     │
│  - Extract links, attachments                                        │
│  - Group thread replies                                              │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                   Send to MycelicMemory                              │
│  POST /api/v1/sources/{source_id}/ingest                             │
│  - Batch of IngestItems                                              │
│  - Checkpoint for resume                                             │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Update Sync State                               │
│  - New last_sync_ts per channel                                      │
│  - Log sync history                                                  │
└─────────────────────────────────────────────────────────────────────┘
```

### Backfill Process

```
Initial setup / manual trigger:
1. User configures backfill_days (e.g., 90)
2. For each channel:
   a. conversations.history with oldest param
   b. Paginate through history
   c. Transform and ingest in batches
   d. Update checkpoint after each batch
3. Transition to incremental mode
```

## Desktop App Updates

### New Data Sources Page

```typescript
// src/renderer/pages/DataSources.tsx
interface DataSourcesPage {
  // List all registered sources
  sources: DataSource[];

  // Source management
  registerSource(type: string, config: SourceConfig): void;
  pauseSource(id: string): void;
  resumeSource(id: string): void;
  deleteSource(id: string): void;

  // Sync control
  triggerSync(id: string): void;
  viewSyncHistory(id: string): SyncHistoryEntry[];

  // Configuration
  editSourceConfig(id: string, config: SourceConfig): void;
}
```

### Source-specific Configuration UI

```typescript
// Slack configuration component
interface SlackConfigPanel {
  // OAuth flow
  connectSlack(): void;  // Opens OAuth popup

  // Channel selection
  channels: Channel[];
  selectedChannels: string[];

  // DM selection
  dms: DirectMessage[];
  selectedDMs: string[];

  // Sync settings
  intervalMinutes: number;
  backfillDays: number;
  filters: FilterConfig;
}
```

## Database Schema Extensions

```sql
-- Source registry
CREATE TABLE IF NOT EXISTS data_sources (
    id TEXT PRIMARY KEY,
    source_type TEXT NOT NULL,
    name TEXT NOT NULL,
    config TEXT NOT NULL,  -- JSON
    status TEXT DEFAULT 'active',
    last_sync_at DATETIME,
    last_sync_position TEXT,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Sync history
CREATE TABLE IF NOT EXISTS data_source_sync_history (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    items_processed INTEGER DEFAULT 0,
    memories_created INTEGER DEFAULT 0,
    status TEXT DEFAULT 'running',
    error TEXT,
    FOREIGN KEY (source_id) REFERENCES data_sources(id)
);

-- Track which memories came from which source
ALTER TABLE memories ADD COLUMN source_id TEXT;
ALTER TABLE memories ADD COLUMN external_id TEXT;

CREATE INDEX idx_memories_source ON memories(source_id);
CREATE UNIQUE INDEX idx_memories_external ON memories(source_id, external_id);
```

## API Endpoints to Add

```yaml
# Source Management
POST   /api/v1/sources                    # Register new source
GET    /api/v1/sources                    # List all sources
GET    /api/v1/sources/{id}               # Get source details
PATCH  /api/v1/sources/{id}               # Update source config
DELETE /api/v1/sources/{id}               # Remove source

# Source Control
POST   /api/v1/sources/{id}/pause         # Pause syncing
POST   /api/v1/sources/{id}/resume        # Resume syncing
POST   /api/v1/sources/{id}/sync          # Trigger manual sync

# Ingestion
POST   /api/v1/sources/{id}/ingest        # Bulk ingest items

# History
GET    /api/v1/sources/{id}/history       # Sync history
GET    /api/v1/sources/{id}/stats         # Source statistics
```

## Implementation Priority

### Phase 1: Core Infrastructure (Required First)
1. Add `data_sources` and `data_source_sync_history` tables
2. Add source_id column to memories
3. Implement source registry API endpoints
4. Implement unified ingestion endpoint
5. Update desktop app with Data Sources page

### Phase 2: Slack Exporter Service
1. Create standalone Node.js service
2. Implement Slack OAuth2 flow
3. Implement incremental sync
4. Implement backfill
5. Add configuration UI in desktop app

### Phase 3: Future Sources
- Email (IMAP/Gmail API)
- Browser history (via extension)
- Notion (API)
- Obsidian (local vault scanning)
- GitHub (issues, PRs, discussions)

## Security Considerations

1. **Token Storage**: All API tokens should be encrypted at rest
2. **OAuth Scopes**: Request minimum required scopes
3. **Rate Limiting**: Respect API rate limits to avoid bans
4. **Data Privacy**: Allow filtering out sensitive content
5. **Access Control**: Source configurations should be user-specific

## Configuration Example

```json
{
  "sources": [
    {
      "id": "slack-work",
      "type": "slack",
      "name": "Work Slack",
      "config": {
        "channels": {
          "mode": "include",
          "list": ["#engineering", "#product", "#general"]
        },
        "dms": {
          "enabled": true,
          "mode": "all"
        },
        "sync": {
          "interval_minutes": 60,
          "backfill_days": 90
        },
        "filters": {
          "min_message_length": 20,
          "exclude_bot_messages": true
        }
      }
    },
    {
      "id": "claude-stream",
      "type": "claude-stream",
      "name": "Claude Code Sessions",
      "config": {
        "db_path": "~/.local/share/claude-chat-stream/data/chats.db",
        "auto_extract": true,
        "poll_interval_ms": 5000
      }
    }
  ]
}
```

## Summary

This architecture enables MycelicMemory to scale from a single data source (claude-chat-stream) to multiple continuous data streams by:

1. **Abstracting data ingestion** through a unified API
2. **Registering sources** in a central registry
3. **Tracking sync state** for incremental updates
4. **Supporting backfill** for historical data
5. **Providing configuration UI** in the desktop app

The Slack exporter would be a standalone service that runs independently and pushes data to MycelicMemory through the ingestion API, following the same pattern that future data sources would use.

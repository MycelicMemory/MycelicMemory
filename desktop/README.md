# MycelicMemory Desktop

Desktop application for MycelicMemory - Browse memories, visualize knowledge graphs, and manage services.

## Features

- **Dashboard**: Overview of memory statistics, service health, quick actions, and recent activity
- **Memory Browser**: Search, filter, edit, and delete memories
- **Claude Sessions**: Browse sessions and messages from claude-chat-stream
- **Knowledge Graph**: Visualize memory relationships using vis-network
- **Settings**: Configure API endpoints, models, and preferences

## Prerequisites

- Node.js 18+
- MycelicMemory running on port 3099
- claude-chat-stream database (optional, for session browsing)
- Ollama (optional, for semantic search)
- Qdrant (optional, for vector search)

## Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Create distributable packages
npm run package        # Current platform
npm run package:win    # Windows
npm run package:mac    # macOS
npm run package:linux  # Linux
```

## Architecture

```
src/
├── main/              # Electron main process
│   ├── index.ts       # Entry point, window management
│   ├── preload.ts     # Context bridge for renderer
│   ├── ipc/           # IPC handlers
│   │   ├── memory.ipc.ts
│   │   ├── claude.ipc.ts
│   │   └── config.ipc.ts
│   └── services/      # Backend services
│       ├── mycelicmemory-client.ts  # REST API wrapper
│       └── claude-stream-db.ts      # SQLite reader
│
├── renderer/          # React frontend
│   ├── App.tsx        # Main app with routing
│   ├── pages/         # Page components
│   │   ├── Dashboard.tsx
│   │   ├── MemoryBrowser.tsx
│   │   ├── ClaudeSessions.tsx
│   │   ├── KnowledgeGraph.tsx
│   │   └── Settings.tsx
│   └── styles/        # CSS styles
│
└── shared/            # Shared TypeScript types
    └── types.ts
```

## Database Paths

| Database | Platform | Path |
|----------|----------|------|
| MycelicMemory | All | `~/.mycelicmemory/memories.db` |
| Claude Chat Stream | Windows | `%LOCALAPPDATA%/claude-chat-stream/data/chats.db` |
| Claude Chat Stream | macOS | `~/Library/Application Support/claude-chat-stream/data/chats.db` |
| Claude Chat Stream | Linux | `~/.config/claude-chat-stream/data/chats.db` |

## License

MIT

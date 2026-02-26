# MycelicMemory Desktop UI: Comprehensive Improvement Plan

## Philosophy

Each layer builds on the previous. Improvements are **recursively decomposed** — no item is a monolith. Every change is a discrete, testable, shippable unit that unlocks the next improvement.

---

## Layer 0: Foundation Infrastructure (Enables Everything Else)

These are invisible to the user but make every subsequent improvement possible.

### 0.1 Global State Management with Zustand
**Why first**: Every page currently uses independent `useState` calls with no sharing. Dashboard, MemoryBrowser, Graph, and Sessions all fetch the same data independently. Without shared state, no improvement can coordinate across pages.

- **0.1.1** Install zustand (~2KB, no boilerplate)
- **0.1.2** Create `renderer/stores/memoryStore.ts`
  - State: `memories[]`, `totalCount`, `loading`, `error`, `filters`, `selectedId`
  - Actions: `fetchMemories()`, `createMemory()`, `updateMemory()`, `deleteMemory()`, `searchMemories()`
  - Selectors: `useMemories()`, `useMemoryById(id)`, `useMemoryCount()`
- **0.1.3** Create `renderer/stores/sessionStore.ts`
  - State: `projects[]`, `sessions[]`, `selectedProject`, `selectedSession`, `messages[]`, `toolCalls[]`
  - Actions: `fetchProjects()`, `fetchSessions()`, `fetchMessages()`, `ingestConversations()`
- **0.1.4** Create `renderer/stores/domainStore.ts`
  - State: `domains[]`, `domainStats{}`, `loading`
  - Actions: `fetchDomains()`, `fetchDomainStats(domain)`, `createDomain()`
- **0.1.5** Create `renderer/stores/relationshipStore.ts`
  - State: `relationships{}`, `graphData`, `loading`
  - Actions: `fetchRelationships(memoryId)`, `createRelationship()`, `discoverRelationships()`
- **0.1.6** Create `renderer/stores/sourceStore.ts`
  - State: `sources[]`, `syncHistory{}`, `activeJobs{}`, `loading`
  - Actions: `fetchSources()`, `createSource()`, `triggerSync()`, `pauseSource()`, `resumeSource()`
- **0.1.7** Create `renderer/stores/uiStore.ts`
  - State: `sidebarCollapsed`, `theme`, `commandPaletteOpen`, `activeModal`, `toastQueue`
  - Actions: `toggleSidebar()`, `setTheme()`, `openModal()`, `closeModal()`
- **0.1.8** Create `renderer/stores/settingsStore.ts`
  - State: `settings{}`, `health{}`, `connected`, `ollamaModels[]`
  - Actions: `fetchSettings()`, `updateSettings()`, `testConnections()`
- **0.1.9** Migrate Dashboard.tsx to use stores (remove local state)
- **0.1.10** Migrate MemoryBrowser.tsx to use stores
- **0.1.11** Migrate ClaudeSessions.tsx to use stores
- **0.1.12** Migrate KnowledgeGraph.tsx to use stores
- **0.1.13** Migrate Settings.tsx to use stores

### 0.2 API Client Layer Consolidation
**Why**: api-bridge.ts is a flat object with no error normalization, no retry, no caching. Every page re-implements error handling.

- **0.2.1** Create `renderer/lib/apiClient.ts` — centralized fetch wrapper
  - Automatic response unwrapping (`data.data ?? data`)
  - Retry with exponential backoff (3 attempts for 5xx)
  - Request deduplication (same URL within 100ms → reuse promise)
  - Error normalization → `{code, message, details}` shape
  - Request/response interceptors for logging
- **0.2.2** Add all missing endpoints to api-bridge.ts:
  - `POST /api/v1/relationships` (create relationship)
  - `POST /api/v1/analyze` (AI analysis)
  - `POST /api/v1/categories` (create category)
  - `GET /api/v1/categories` (list categories)
  - `POST /api/v1/memories/:id/categorize` (categorize memory)
  - `GET /api/v1/categories/stats` (category stats)
  - `POST /api/v1/domains` (create domain)
  - `GET /api/v1/domains/:domain/stats` (domain stats)
  - `POST /api/v1/search/tags` (tag search)
  - `POST /api/v1/search/date-range` (date range search)
  - `POST /api/v1/memories/search/intelligent` (intelligent search)
  - `GET /api/v1/memories/:id/related` (related memories)
  - `GET /api/v1/memories/:id/trace` (source tracing)
  - Full data sources CRUD: `POST/GET/PATCH/DELETE /api/v1/sources`
  - Source operations: `pause`, `resume`, `sync`, `ingest`, `history`, `stats`, `memories`
- **0.2.3** Add TypeScript interfaces for all request/response shapes
- **0.2.4** Wire stores to use apiClient instead of raw fetch

### 0.3 Tailwind Theme System
**Why**: Theme toggle exists in Settings but CSS is hardcoded dark. Light mode impossible without proper theme tokens.

- **0.3.1** Add `darkMode: 'class'` to `tailwind.config.js`
- **0.3.2** Define CSS custom properties in `index.css`:
  - `--bg-primary`, `--bg-secondary`, `--bg-tertiary`
  - `--text-primary`, `--text-secondary`, `--text-muted`
  - `--border`, `--border-hover`
  - `--accent`, `--accent-hover`
  - `--success`, `--warning`, `--error`
  - `--card-bg`, `--card-border`
- **0.3.3** Create `.dark` and `.light` theme classes
- **0.3.4** Wire theme from uiStore → `<html>` class toggle
- **0.3.5** Replace all hardcoded `bg-slate-*`, `text-slate-*` with theme-aware classes across:
  - App.tsx (sidebar, header, main area)
  - Dashboard.tsx (stat cards, charts, service panel)
  - MemoryBrowser.tsx (list, detail panel, filters)
  - ClaudeSessions.tsx (three panels, messages)
  - KnowledgeGraph.tsx (controls, detail panel)
  - Settings.tsx (sections, inputs, code blocks)
  - All shared components
- **0.3.6** Verify system theme detection (`prefers-color-scheme`) for "System" option

---

## Layer 1: Fix Broken/Fake Data (Trust the UI)

Users see fake numbers. Fix these before adding features.

### 1.1 Dashboard Real Data
- **1.1.1** Fix "This Week" count: Query memories where `created_at > 7 days ago`
  - Backend: Add `created_after` parameter to `GET /api/v1/memories` or use date-range search
  - Frontend: Call with date filter, use count from response
- **1.1.2** Fix domain memory counts: Call `GET /api/v1/domains/:domain/stats` for each domain
  - Currently uses `Math.random()` (line 236 of Dashboard.tsx)
  - Replace with actual `memory_count` from domain stats
- **1.1.3** Fix importance distribution: Query actual importance breakdown
  - Backend: Add `GET /api/v1/stats/importance-distribution` or compute client-side from memory list
  - Currently hardcoded array (lines 239-244)
- **1.1.4** Fix service port display: Use actual port from health response instead of placeholder "3099"
- **1.1.5** Add real "memories created today" count to dashboard

### 1.2 Memory Browser Real Search
- **1.2.1** Add search debounce (300ms) to prevent excessive API calls
- **1.2.2** Wire semantic search toggle to actually pass `use_ai: true` to search endpoint
- **1.2.3** Show search result count and search time from `search_info` response
- **1.2.4** Reset pagination to page 1 when search/filter changes

### 1.3 Knowledge Graph Complete Data
- **1.3.1** Remove 50-memory limit on relationship fetching — use paginated approach:
  - Fetch relationships for all loaded memories, not just first 50
  - Use batch endpoint or parallel requests with concurrency limit (5)
- **1.3.2** Show relationship strength visually (edge thickness or opacity)
- **1.3.3** Load actual node count in graph toolbar instead of estimating

---

## Layer 2: Expose Hidden Backend Capabilities (Unlock 60% More Features)

The backend has 29 endpoints. The desktop UI only uses ~15. These additions use existing APIs.

### 2.1 Data Sources Management Page (NEW PAGE)
**Unlocks**: The entire multi-source ingestion system. Currently invisible to users.

- **2.1.1** Create `renderer/pages/DataSources.tsx`
- **2.1.2** Add "/sources" route to App.tsx with sidebar icon (Database icon)
- **2.1.3** Source List Panel:
  - Table/grid view of all registered data sources
  - Columns: Name, Type, Status (active/paused/error), Last Sync, Memory Count
  - Status badge with color coding (green=active, yellow=paused, red=error)
  - Inline actions: Sync, Pause/Resume, Edit, Delete
- **2.1.4** Add Source Dialog:
  - Source type selector (dropdown): claude-code-local, slack, discord, telegram, imessage, email, browser, notion, obsidian, github, custom
  - Name input
  - Config JSON editor (per-type template)
  - Validate button (calls adapter.Validate())
- **2.1.5** Source Detail Panel (click to expand):
  - Stats: total memories, sync count, success rate, last error
  - Sync History timeline: each sync with status, items processed, duration
  - Memory list from this source (paginated)
  - Config viewer/editor
- **2.1.6** Manual Sync Button:
  - Calls `POST /api/v1/sources/:id/sync`
  - Shows progress indicator
  - Auto-refreshes stats on completion
- **2.1.7** Backfill Mode Toggle:
  - Full reprocess vs incremental
  - Warning dialog for backfill (may create duplicates if dedup logic changes)

### 2.2 Categories System
- **2.2.1** Add category display to memory cards in MemoryBrowser
- **2.2.2** Create Category sidebar filter in MemoryBrowser
- **2.2.3** Create Category management section in Settings or as sub-page
  - List categories with memory counts
  - Create/edit/delete categories
  - Hierarchical display (parent-child)
- **2.2.4** Add "Categorize" button to memory detail panel
  - Dropdown to select category
  - Auto-categorize button (AI, if Ollama available)

### 2.3 Related Memories
- **2.3.1** Add "Related" tab to memory detail panel in MemoryBrowser
  - Calls `GET /api/v1/memories/:id/related`
  - Shows related memories with relationship type and strength
- **2.3.2** Click related memory to navigate to it
- **2.3.3** "Create Relationship" button in related panel
  - Memory search/select for target
  - Relationship type dropdown (7 types)
  - Strength slider (0.0-1.0)
  - Context text input

### 2.4 Source Tracing
- **2.4.1** Add "Source" indicator to memory cards that have `cc_session_id`
  - Small link icon or badge
- **2.4.2** Click source indicator → navigate to conversation in ClaudeSessions
- **2.4.3** In memory detail panel, show "Originated from" section with session info
  - Calls `GET /api/v1/memories/:id/trace`
  - Shows: session title, date, project, message count

### 2.5 AI Analysis Panel
- **2.5.1** Create `renderer/components/AnalysisPanel.tsx`
  - Text input for questions
  - Analysis type selector: Question, Summarize, Analyze, Temporal Patterns
  - Timeframe filter: Today, Week, Month, All
  - Domain filter
  - Results display with markdown rendering
- **2.5.2** Add Analysis as tab in Dashboard
- **2.5.3** Add "Analyze" quick action in memory detail (pre-fills with memory content)

### 2.6 Advanced Search
- **2.6.1** Add search mode tabs in MemoryBrowser: Keyword | Semantic | Tags | Date Range | Intelligent
- **2.6.2** Tag search mode:
  - Tag input with autocomplete from existing tags
  - AND/OR toggle
  - Results filtered by selected tags
- **2.6.3** Date range search mode:
  - Date picker (start/end)
  - Quick presets: Today, This Week, This Month, Last 30 Days
- **2.6.4** Intelligent search mode:
  - Single search box that combines FTS5 + semantic + AI
  - Shows search strategy explanation
  - Requires Ollama (show disabled state if unavailable)

### 2.7 Domain Management
- **2.7.1** Add domain creation to MemoryBrowser filter panel
  - "Create Domain" button with name + description inputs
- **2.7.2** Show domain stats inline (memory count, avg importance)
- **2.7.3** Allow domain editing from memory detail panel
  - Change memory's domain via dropdown
  - Create new domain inline

---

## Layer 3: UI Polish & Interaction Quality (Make It Feel Professional)

### 3.1 Loading States
- **3.1.1** Create `renderer/components/Skeleton.tsx` — reusable skeleton components
  - SkeletonCard (memory card shape)
  - SkeletonLine (text line)
  - SkeletonChart (chart placeholder)
  - SkeletonTable (table rows)
- **3.1.2** Replace all spinner-only loading states:
  - Dashboard: skeleton stat cards + skeleton charts
  - MemoryBrowser: skeleton memory cards in list
  - ClaudeSessions: skeleton project/session/message items
  - KnowledgeGraph: skeleton canvas with "Loading graph..." overlay
  - Settings: skeleton form fields

### 3.2 Content Rendering
- **3.2.1** Install `react-markdown` + `react-syntax-highlighter`
- **3.2.2** Create `renderer/components/MarkdownContent.tsx`
  - Renders markdown with syntax highlighting
  - Code blocks with language detection and copy button
  - Links open in external browser
  - Tables rendered properly
- **3.2.3** Apply to memory content display in MemoryBrowser detail panel
- **3.2.4** Apply to message content in ClaudeSessions
- **3.2.5** Apply to analysis results

### 3.3 Keyboard Shortcuts
- **3.3.1** Create `renderer/hooks/useHotkeys.ts` — centralized hotkey manager
- **3.3.2** Define shortcut map:
  - `Ctrl+K` → Command Palette (exists)
  - `Ctrl+N` → Create Memory (exists)
  - `Ctrl+B` → Toggle Sidebar (exists)
  - `Ctrl+/` → Show shortcuts help
  - `Ctrl+1-5` → Navigate to page (Dashboard, Memories, Sessions, Graph, Settings)
  - `Escape` → Close modal/panel/palette
  - `j/k` → Navigate list items (vim-style)
  - `Enter` → Select/open item
  - `d` → Delete selected (with confirmation)
  - `e` → Edit selected
  - `r` → Refresh current page data
  - `f` → Focus search input
  - `/` → Focus search in current page
- **3.3.3** Create `renderer/components/ShortcutsHelp.tsx` — modal showing all shortcuts
- **3.3.4** Add "?" icon in sidebar footer to open shortcuts help

### 3.4 Responsive Layout
- **3.4.1** Make sidebar responsive: auto-collapse below 1024px
- **3.4.2** MemoryBrowser: stack list and detail vertically on narrow screens
- **3.4.3** ClaudeSessions: stack panels vertically on narrow screens
- **3.4.4** KnowledgeGraph: detail panel as bottom drawer on narrow screens
- **3.4.5** Dashboard: responsive grid (2 columns → 1 column on narrow)

### 3.5 Animations & Transitions
- **3.5.1** Page transitions: fade-in on route change (existing, verify working)
- **3.5.2** Sidebar collapse animation: smooth width transition (0.2s ease)
- **3.5.3** Modal open/close: scale + fade animation
- **3.5.4** Toast enter/exit: slide from right
- **3.5.5** List item hover: subtle background transition
- **3.5.6** Stat card hover: slight lift shadow
- **3.5.7** Graph node hover: glow effect

---

## Layer 4: Data Operations & Export (Power User Features)

### 4.1 Bulk Operations
- **4.1.1** Add checkbox to memory cards in MemoryBrowser list
- **4.1.2** "Select All" checkbox in list header
- **4.1.3** Selection count indicator in toolbar: "3 selected"
- **4.1.4** Bulk actions dropdown when items selected:
  - Delete selected (with count confirmation)
  - Change domain (domain picker)
  - Change importance (slider)
  - Add tags (tag input)
  - Export selected
- **4.1.5** Progress indicator for bulk operations

### 4.2 Export Features
- **4.2.1** Create `renderer/utils/export.ts` — export utilities
- **4.2.2** Memory export formats:
  - JSON (full data with relationships)
  - CSV (flat fields)
  - Markdown (formatted for reading)
- **4.2.3** Session export:
  - Markdown (conversation format with tool calls)
  - JSON (full session with messages and tool calls)
- **4.2.4** Graph export:
  - PNG/SVG screenshot
  - JSON (nodes + edges)
  - DOT format (for Graphviz)
- **4.2.5** Export buttons:
  - MemoryBrowser toolbar: "Export" dropdown
  - ClaudeSessions toolbar: "Export Session" button
  - KnowledgeGraph toolbar: "Export Graph" button
  - Dashboard: "Export All Memories" in quick actions

### 4.3 Import Features
- **4.3.1** Memory import from JSON
  - File picker
  - Preview imported items
  - Deduplication check
  - Import with progress bar
- **4.3.2** Memory import from Markdown (parse headings as separate memories)

---

## Layer 5: Session & Conversation Enhancements

### 5.1 Session Detail Improvements
- **5.1.1** Add message search within session (search across all messages)
- **5.1.2** Render tool call input JSON with syntax highlighting
- **5.1.3** Render tool call output/result text
- **5.1.4** Add timestamps to each message
- **5.1.5** Add token count display per message
- **5.1.6** Add session statistics header:
  - Duration (first to last message)
  - Total tokens (sum of estimates)
  - Tool calls by type (pie chart)
  - Files touched (unique file paths from tool calls)
- **5.1.7** Message markdown rendering (code blocks, links, lists)
- **5.1.8** Collapsible tool call sections (expand to see input/output)

### 5.2 Session Filtering
- **5.2.1** Date range filter for sessions
- **5.2.2** Model filter dropdown
- **5.2.3** Sort by: Date, Message Count, Tool Call Count
- **5.2.4** Session pagination (currently loads all)

### 5.3 Session Timeline
- **5.3.1** Add timeline view option (alternative to list view)
  - Horizontal timeline with session dots
  - Hover for preview
  - Click to select
  - Color-coded by project

---

## Layer 6: Knowledge Graph Enhancements

### 6.1 Performance
- **6.1.1** Progressive loading: start with top 50 most-connected nodes, expand on demand
- **6.1.2** Virtual scrolling for node detail lists
- **6.1.3** Web Worker for graph layout computation (prevent UI thread blocking)
- **6.1.4** Lazy relationship loading: fetch relationships as nodes enter viewport

### 6.2 Interaction
- **6.2.1** Double-click node → open memory in MemoryBrowser
- **6.2.2** Right-click context menu:
  - View details
  - Find related
  - Create relationship to...
  - Navigate to source
  - Delete (with confirmation)
- **6.2.3** Drag to create edge: drag from one node to another → create relationship dialog
- **6.2.4** Search nodes: text input that highlights matching nodes and centers view
- **6.2.5** Node grouping by domain with cluster visualization

### 6.3 Visual Improvements
- **6.3.1** Edge thickness based on relationship strength
- **6.3.2** Edge color based on relationship type (7 distinct colors)
- **6.3.3** Node size based on connection count
- **6.3.4** Minimap overview in corner
- **6.3.5** Zoom controls with percentage display

---

## Layer 7: Settings & Configuration Polish

### 7.1 Settings Organization
- **7.1.1** Collapsible sections (Connection, AI/Models, Appearance, Data, Advanced)
- **7.1.2** Section navigation: sticky sidebar within settings page
- **7.1.3** Unsaved changes indicator + "Discard Changes" button
- **7.1.4** "Reset to Defaults" button per section

### 7.2 Validation
- **7.2.1** API URL format validation (must be valid URL)
- **7.2.2** Port range validation (1-65535)
- **7.2.3** Ollama URL validation with live test
- **7.2.4** Database path validation (check existence)
- **7.2.5** Inline validation messages (not just on save)

### 7.3 Configuration Export/Import
- **7.3.1** Export settings as JSON
- **7.3.2** Import settings from JSON
- **7.3.3** Settings backup before changes

---

## Layer 8: Command Palette Enhancement (Builds on Layer 0)

### 8.1 Search Quality
- **8.1.1** Fuzzy matching (fuse.js) for command palette results
- **8.1.2** Search history: show recent searches when palette opens empty
- **8.1.3** Categorized results with section headers
- **8.1.4** Result previews: show memory content snippet on highlight

### 8.2 Actions
- **8.2.1** Add command actions (not just search):
  - "Create Memory" → opens modal
  - "Go to Dashboard" → navigates
  - "Go to Memories" → navigates
  - "Go to Sessions" → navigates
  - "Go to Graph" → navigates
  - "Go to Sources" → navigates
  - "Go to Settings" → navigates
  - "Toggle Theme" → switches dark/light
  - "Toggle Sidebar" → collapses/expands
  - "Ingest Conversations" → triggers ingest
  - "Export Memories" → opens export dialog
- **8.2.2** ">" prefix for commands (like VS Code)
- **8.2.3** "?" prefix for help/documentation

---

## Layer 9: Notification & Feedback System (Builds on Layers 0-2)

### 9.1 Progress Tracking
- **9.1.1** Create `renderer/components/ProgressTracker.tsx`
  - Shows long-running operations in a bottom panel
  - Ingestion progress with items/total, sessions, memories
  - Relationship discovery progress
  - Bulk operation progress
- **9.1.2** Wire to pipeline progress channel (WebSocket or polling)
- **9.1.3** Notification badge in sidebar for active operations

### 9.2 Real-time Updates
- **9.2.1** Auto-refresh memory list when ingestion completes
- **9.2.2** Auto-refresh graph when relationships discovered
- **9.2.3** Auto-refresh dashboard stats after any mutation
- **9.2.4** Subtle "data updated" indicator (not disruptive)

---

## Implementation Sequence

The layers should be implemented roughly in order, but within each layer, items can be parallelized:

```
Layer 0 (Foundation)     ─── 3-4 hours
  └─► Layer 1 (Fix Data) ─── 1-2 hours
       └─► Layer 2 (Expose APIs) ─── 4-6 hours
            ├─► Layer 3 (Polish) ─── 3-4 hours
            ├─► Layer 4 (Data Ops) ─── 2-3 hours
            └─► Layer 5 (Sessions) ─── 2-3 hours
                 └─► Layer 6 (Graph) ─── 3-4 hours
                      └─► Layer 7 (Settings) ─── 1-2 hours
                           └─► Layer 8 (Palette) ─── 1-2 hours
                                └─► Layer 9 (Notifications) ─── 2-3 hours
```

**Total: ~115 discrete improvements across 10 layers**

## What To Implement Next (Recommended Priority)

**Batch 1** (Maximum Impact, Minimum Effort):
1. Layer 1.1 (Fix Dashboard fake data) — trust the UI
2. Layer 0.3 (Theme system) — visible, satisfying
3. Layer 2.1 (Data Sources page) — unlocks multi-source
4. Layer 3.2 (Markdown rendering) — dramatically improves content display

**Batch 2** (Deep Capability):
5. Layer 0.1 + 0.2 (State management + API client) — enables everything
6. Layer 2.3 (Related memories) — graph becomes interactive
7. Layer 2.6 (Advanced search) — power user enablement
8. Layer 3.1 (Loading skeletons) — professional feel

**Batch 3** (Polish & Power):
9. Layer 4 (Bulk ops + export)
10. Layer 5 (Session enhancements)
11. Layer 6 (Graph enhancements)
12. Remaining layers

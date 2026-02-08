import { useState, useEffect, useRef, useCallback } from 'react';
import { Network, RefreshCw, ZoomIn, ZoomOut, Maximize2, MessageSquare } from 'lucide-react';
import type { Memory, MemoryRelationship, ClaudeSession } from '../../shared/types';

// Note: vis-network types
interface NetworkNode {
  id: string;
  label: string;
  title?: string;
  color?: string | { background: string; border: string };
  shape?: string;
  size?: number;
  font?: { color?: string; size?: number };
}

interface NetworkEdge {
  id: string;
  from: string;
  to: string;
  label?: string;
  color?: string | { color: string };
  width?: number;
  arrows?: string;
  dashes?: boolean;
}

const RELATIONSHIP_COLORS: Record<string, string> = {
  references: '#6366f1',
  contradicts: '#ef4444',
  expands: '#22c55e',
  similar: '#8b5cf6',
  sequential: '#f59e0b',
  causes: '#06b6d4',
  enables: '#ec4899',
};

const DOMAIN_COLORS: Record<string, string> = {
  general: '#6366f1',
  frontend: '#22c55e',
  backend: '#f59e0b',
  database: '#06b6d4',
  devops: '#8b5cf6',
  testing: '#ec4899',
  programming: '#6366f1',
  code: '#14b8a6',
  conversations: '#f97316',
};

// Distinct colors for session nodes by project
const SESSION_COLORS = [
  '#f97316', // orange
  '#06b6d4', // cyan
  '#a855f7', // purple
  '#ec4899', // pink
  '#84cc16', // lime
  '#eab308', // yellow
  '#14b8a6', // teal
  '#f43f5e', // rose
];

const SESSION_NODE_COLOR = '#f97316'; // orange for all session nodes
const SOURCE_EDGE_COLOR = '#f97316'; // orange dashed edge for session links

type SelectedItem =
  | { type: 'memory'; data: Memory }
  | { type: 'session'; data: ClaudeSession };

export default function KnowledgeGraph() {
  const containerRef = useRef<HTMLDivElement>(null);
  const networkRef = useRef<any>(null);
  const [memories, setMemories] = useState<Memory[]>([]);
  const [relationships, setRelationships] = useState<MemoryRelationship[]>([]);
  const [sessions, setSessions] = useState<ClaudeSession[]>([]);
  const [selectedItem, setSelectedItem] = useState<SelectedItem | null>(null);
  const [loading, setLoading] = useState(true);
  const [discovering, setDiscovering] = useState(false);
  const [showSessions, setShowSessions] = useState(true);
  const [filter, setFilter] = useState({
    domain: '',
    relationshipType: '',
  });

  useEffect(() => {
    fetchData();
  }, []);

  useEffect(() => {
    if (memories.length > 0 && containerRef.current) {
      initializeNetwork();
    }
    return () => {
      if (networkRef.current) {
        networkRef.current.destroy();
        networkRef.current = null;
      }
    };
  }, [memories, relationships, sessions, filter, showSessions]);

  async function fetchData() {
    try {
      setLoading(true);
      const [memoriesRes, relationsRes, sessionsRes] = await Promise.all([
        window.mycelicMemory.memory.list({ limit: 200 }),
        window.mycelicMemory.relationships.discover().catch(() => []),
        window.mycelicMemory.claude.sessions().catch(() => []),
      ]);
      setMemories(memoriesRes || []);
      setRelationships(relationsRes || []);
      setSessions(sessionsRes || []);
    } catch (err) {
      console.error('Failed to fetch data:', err);
    } finally {
      setLoading(false);
    }
  }

  async function handleDiscoverRelationships() {
    try {
      setDiscovering(true);
      const newRelations = await window.mycelicMemory.relationships.discover();
      setRelationships(newRelations);
    } catch (err) {
      console.error('Failed to discover relationships:', err);
    } finally {
      setDiscovering(false);
    }
  }

  const initializeNetwork = useCallback(async () => {
    if (!containerRef.current) return;

    const vis = await import('vis-network/standalone');

    // Filter memories
    let filteredMemories = memories;
    let filteredRelationships = relationships;

    if (filter.domain) {
      filteredMemories = memories.filter((m) => m.domain === filter.domain);
      const memoryIds = new Set(filteredMemories.map((m) => m.id));
      filteredRelationships = relationships.filter(
        (r) => memoryIds.has(r.source_id) && memoryIds.has(r.target_id)
      );
    }

    if (filter.relationshipType) {
      filteredRelationships = filteredRelationships.filter(
        (r) => r.relationship_type === filter.relationshipType
      );
    }

    // Build memory nodes
    const nodes: NetworkNode[] = filteredMemories.map((memory) => {
      const isSessionSummary = memory.source === 'claude-code-session';
      return {
        id: memory.id,
        label: memory.content.substring(0, 30) + (memory.content.length > 30 ? '...' : ''),
        title: memory.content,
        color: {
          background: isSessionSummary
            ? SESSION_NODE_COLOR
            : DOMAIN_COLORS[memory.domain || 'general'] || DOMAIN_COLORS.general,
          border: isSessionSummary ? '#ea580c' : '#1e293b',
        },
        shape: isSessionSummary ? 'diamond' : 'dot',
        size: isSessionSummary ? 18 : 10 + memory.importance * 2,
      };
    });

    // Build session nodes (if enabled)
    const sessionNodeMap = new Map<string, ClaudeSession>();
    if (showSessions) {
      // Find which sessions are referenced by memories
      const referencedSessionIds = new Set(
        filteredMemories
          .filter((m) => m.cc_session_id)
          .map((m) => m.cc_session_id!)
      );

      // Build a project color map
      const projectColors = new Map<string, string>();
      let colorIdx = 0;

      sessions.forEach((s) => {
        if (referencedSessionIds.has(s.id)) {
          sessionNodeMap.set(s.id, s);

          if (!projectColors.has(s.project_hash)) {
            projectColors.set(s.project_hash, SESSION_COLORS[colorIdx % SESSION_COLORS.length]);
            colorIdx++;
          }

          const projectColor = projectColors.get(s.project_hash)!;
          const projectName = s.project_path.split(/[/\\]/).pop() || s.project_hash;
          const label = s.title
            ? s.title.substring(0, 25) + (s.title.length > 25 ? '...' : '')
            : `Chat ${s.session_id.substring(0, 8)}`;

          nodes.push({
            id: `session:${s.id}`,
            label,
            title: `[${projectName}] ${s.title || s.first_prompt || 'Chat session'}\n${s.message_count} messages, ${s.tool_call_count} tool calls\n${new Date(s.created_at).toLocaleDateString()}`,
            color: {
              background: projectColor,
              border: '#1e293b',
            },
            shape: 'star',
            size: 20,
            font: { color: '#f1f5f9', size: 11 },
          });
        }
      });
    }

    // Build relationship edges
    const edges: NetworkEdge[] = filteredRelationships.map((rel) => ({
      id: rel.id,
      from: rel.source_id,
      to: rel.target_id,
      label: rel.relationship_type,
      color: {
        color: RELATIONSHIP_COLORS[rel.relationship_type] || '#6366f1',
      },
      width: rel.strength * 3,
      arrows: 'to',
    }));

    // Build session-to-memory edges (dashed orange)
    if (showSessions) {
      filteredMemories.forEach((memory) => {
        if (memory.cc_session_id && sessionNodeMap.has(memory.cc_session_id)) {
          edges.push({
            id: `trace:${memory.id}`,
            from: memory.id,
            to: `session:${memory.cc_session_id}`,
            color: { color: SOURCE_EDGE_COLOR },
            width: 1.5,
            arrows: 'to',
            dashes: true,
          });
        }
      });
    }

    const data = {
      nodes: new vis.DataSet(nodes),
      edges: new vis.DataSet(edges),
    };

    const options = {
      nodes: {
        font: {
          color: '#f1f5f9',
          size: 12,
        },
        borderWidth: 2,
      },
      edges: {
        font: {
          color: '#94a3b8',
          size: 10,
          strokeWidth: 0,
        },
        smooth: {
          type: 'continuous',
        },
      },
      physics: {
        enabled: true,
        barnesHut: {
          gravitationalConstant: -2000,
          centralGravity: 0.3,
          springLength: 150,
          springConstant: 0.04,
        },
        stabilization: {
          iterations: 100,
        },
      },
      interaction: {
        hover: true,
        tooltipDelay: 200,
        zoomView: true,
        dragView: true,
      },
    };

    if (networkRef.current) {
      networkRef.current.destroy();
    }

    // Re-check after async import
    if (!containerRef.current) return;

    networkRef.current = new vis.Network(containerRef.current, data, options);

    networkRef.current.on('selectNode', (params: { nodes: string[] }) => {
      const nodeId = params.nodes[0];

      // Check if it's a session node
      if (nodeId.startsWith('session:')) {
        const sessId = nodeId.replace('session:', '');
        const session = sessionNodeMap.get(sessId);
        if (session) {
          setSelectedItem({ type: 'session', data: session });
        }
        return;
      }

      // It's a memory node
      const memory = memories.find((m) => m.id === nodeId);
      if (memory) {
        setSelectedItem({ type: 'memory', data: memory });
      }
    });

    networkRef.current.on('deselectNode', () => {
      setSelectedItem(null);
    });
  }, [memories, relationships, sessions, filter, showSessions]);

  function handleZoomIn() {
    if (networkRef.current) {
      const scale = networkRef.current.getScale();
      networkRef.current.moveTo({ scale: scale * 1.2 });
    }
  }

  function handleZoomOut() {
    if (networkRef.current) {
      const scale = networkRef.current.getScale();
      networkRef.current.moveTo({ scale: scale / 1.2 });
    }
  }

  function handleFit() {
    if (networkRef.current) {
      networkRef.current.fit();
    }
  }

  const domains = [...new Set(memories.map((m) => m.domain || 'general'))];
  const relationshipTypes = [...new Set(relationships.map((r) => r.relationship_type))];
  const linkedCount = memories.filter((m) => m.cc_session_id).length;
  const sessionNodeCount = showSessions
    ? new Set(memories.filter((m) => m.cc_session_id).map((m) => m.cc_session_id)).size
    : 0;

  return (
    <div className="h-screen flex flex-col bg-slate-900">
      {/* Toolbar */}
      <div className="p-4 border-b border-slate-700 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h1 className="text-xl font-semibold">Knowledge Graph</h1>
          <span className="text-sm text-slate-400">
            {memories.length} memories
            {sessionNodeCount > 0 && ` + ${sessionNodeCount} chats`}
            {` \u2022 ${relationships.length} connections`}
            {linkedCount > 0 && ` \u2022 ${linkedCount} traced`}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {/* Filters */}
          <select
            value={filter.domain}
            onChange={(e) => setFilter({ ...filter, domain: e.target.value })}
            className="px-3 py-1.5 bg-slate-800 border border-slate-700 rounded-lg text-sm"
          >
            <option value="">All Domains</option>
            {domains.map((d) => (
              <option key={d} value={d}>
                {d}
              </option>
            ))}
          </select>
          <select
            value={filter.relationshipType}
            onChange={(e) => setFilter({ ...filter, relationshipType: e.target.value })}
            className="px-3 py-1.5 bg-slate-800 border border-slate-700 rounded-lg text-sm"
          >
            <option value="">All Relationships</option>
            {relationshipTypes.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>

          <div className="h-6 w-px bg-slate-700 mx-2" />

          {/* Show Sessions Toggle */}
          <button
            onClick={() => setShowSessions(!showSessions)}
            className={`px-3 py-1.5 rounded-lg text-sm flex items-center gap-2 transition-colors ${
              showSessions
                ? 'bg-orange-500/20 text-orange-400 border border-orange-500/50'
                : 'bg-slate-800 text-slate-400 border border-slate-700'
            }`}
            title="Toggle chat session nodes"
          >
            <MessageSquare className="w-4 h-4" />
            Chats
          </button>

          <div className="h-6 w-px bg-slate-700 mx-2" />

          {/* Controls */}
          <button
            onClick={handleZoomIn}
            className="p-2 hover:bg-slate-700 rounded-lg transition-colors"
            title="Zoom In"
          >
            <ZoomIn className="w-4 h-4" />
          </button>
          <button
            onClick={handleZoomOut}
            className="p-2 hover:bg-slate-700 rounded-lg transition-colors"
            title="Zoom Out"
          >
            <ZoomOut className="w-4 h-4" />
          </button>
          <button
            onClick={handleFit}
            className="p-2 hover:bg-slate-700 rounded-lg transition-colors"
            title="Fit to View"
          >
            <Maximize2 className="w-4 h-4" />
          </button>

          <div className="h-6 w-px bg-slate-700 mx-2" />

          <button
            onClick={handleDiscoverRelationships}
            disabled={discovering}
            className="px-3 py-1.5 bg-primary-500/20 text-primary-400 rounded-lg hover:bg-primary-500/30 transition-colors flex items-center gap-2 text-sm disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${discovering ? 'animate-spin' : ''}`} />
            Discover
          </button>
        </div>
      </div>

      <div className="flex-1 flex">
        {/* Graph Container */}
        <div className="flex-1 relative">
          {loading ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="animate-spin w-8 h-8 border-4 border-primary-500 border-t-transparent rounded-full" />
            </div>
          ) : memories.length === 0 ? (
            <div className="absolute inset-0 flex items-center justify-center text-slate-500">
              <div className="text-center">
                <Network className="w-12 h-12 mx-auto mb-4 opacity-50" />
                <p>No memories to visualize</p>
                <p className="text-sm mt-1">Add some memories first</p>
              </div>
            </div>
          ) : (
            <div ref={containerRef} className="w-full h-full" />
          )}
        </div>

        {/* Detail Panel */}
        {selectedItem && (
          <div className="w-80 border-l border-slate-700 p-4 bg-slate-800 overflow-auto">
            {selectedItem.type === 'memory' ? (
              <MemoryDetailPanel memory={selectedItem.data} sessions={sessions} />
            ) : (
              <SessionDetailPanel session={selectedItem.data} />
            )}
          </div>
        )}
      </div>

      {/* Legend */}
      <div className="p-4 border-t border-slate-700 flex items-center gap-6 flex-wrap">
        <span className="text-xs text-slate-400">Domains:</span>
        {Object.entries(DOMAIN_COLORS).slice(0, 6).map(([domain, color]) => (
          <div key={domain} className="flex items-center gap-1.5">
            <div className="w-3 h-3 rounded-full" style={{ backgroundColor: color }} />
            <span className="text-xs text-slate-400">{domain}</span>
          </div>
        ))}
        <div className="h-4 w-px bg-slate-700" />
        <div className="flex items-center gap-1.5">
          <svg width="14" height="14" viewBox="0 0 14 14">
            <polygon points="7,1 13,7 7,13 1,7" fill={SESSION_NODE_COLOR} stroke="#1e293b" strokeWidth="1" />
          </svg>
          <span className="text-xs text-slate-400">session summary</span>
        </div>
        <div className="flex items-center gap-1.5">
          <svg width="14" height="14" viewBox="0 0 14 14">
            <polygon points="7,0 9,5 14,5 10,8 12,14 7,10 2,14 4,8 0,5 5,5" fill={SESSION_NODE_COLOR} stroke="#1e293b" strokeWidth="0.5" />
          </svg>
          <span className="text-xs text-slate-400">chat session</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="w-6 border-t-2 border-dashed" style={{ borderColor: SOURCE_EDGE_COLOR }} />
          <span className="text-xs text-slate-400">traced to</span>
        </div>
      </div>
    </div>
  );
}

function MemoryDetailPanel({ memory, sessions }: { memory: Memory; sessions: ClaudeSession[] }) {
  const linkedSession = memory.cc_session_id
    ? sessions.find((s) => s.id === memory.cc_session_id)
    : null;

  return (
    <>
      <h3 className="font-semibold mb-4">Memory Details</h3>
      <div className="space-y-4">
        <div>
          <label className="text-xs text-slate-400 uppercase">Content</label>
          <p className="mt-1 text-sm">{memory.content}</p>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-xs text-slate-400 uppercase">Domain</label>
            <p className="mt-1 text-sm">{memory.domain || 'general'}</p>
          </div>
          <div>
            <label className="text-xs text-slate-400 uppercase">Importance</label>
            <p className="mt-1 text-sm">{memory.importance}/10</p>
          </div>
        </div>
        {memory.source && (
          <div>
            <label className="text-xs text-slate-400 uppercase">Source</label>
            <p className="mt-1 text-sm">{memory.source}</p>
          </div>
        )}
        {memory.tags && memory.tags.length > 0 && (
          <div>
            <label className="text-xs text-slate-400 uppercase">Tags</label>
            <div className="flex flex-wrap gap-1 mt-1">
              {memory.tags.map((tag) => (
                <span
                  key={tag}
                  className="text-xs bg-primary-500/20 text-primary-400 px-2 py-0.5 rounded"
                >
                  {tag}
                </span>
              ))}
            </div>
          </div>
        )}
        {linkedSession && (
          <div className="pt-3 border-t border-slate-700">
            <label className="text-xs text-orange-400 uppercase">Source Chat Session</label>
            <div className="mt-2 p-3 bg-slate-700/50 rounded-lg">
              <p className="text-sm font-medium">{linkedSession.title || 'Untitled Session'}</p>
              <p className="text-xs text-slate-400 mt-1 line-clamp-2">
                {linkedSession.first_prompt || 'No prompt'}
              </p>
              <div className="flex items-center gap-3 mt-2 text-xs text-slate-400">
                <span>{linkedSession.message_count} msgs</span>
                <span>{linkedSession.tool_call_count} tools</span>
                <span>{new Date(linkedSession.created_at).toLocaleDateString()}</span>
              </div>
              <p className="text-xs text-slate-500 mt-1">
                {linkedSession.project_path.split(/[/\\]/).pop()}
              </p>
            </div>
          </div>
        )}
      </div>
    </>
  );
}

function SessionDetailPanel({ session }: { session: ClaudeSession }) {
  const projectName = session.project_path.split(/[/\\]/).pop() || session.project_hash;

  return (
    <>
      <h3 className="font-semibold mb-4 flex items-center gap-2">
        <MessageSquare className="w-4 h-4 text-orange-400" />
        Chat Session
      </h3>
      <div className="space-y-4">
        <div>
          <label className="text-xs text-slate-400 uppercase">Title</label>
          <p className="mt-1 text-sm">{session.title || 'Untitled Session'}</p>
        </div>
        <div>
          <label className="text-xs text-slate-400 uppercase">First Prompt</label>
          <p className="mt-1 text-sm text-slate-300 line-clamp-4">
            {session.first_prompt || 'No prompt'}
          </p>
        </div>
        <div>
          <label className="text-xs text-slate-400 uppercase">Project</label>
          <p className="mt-1 text-sm">{projectName}</p>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-xs text-slate-400 uppercase">Messages</label>
            <p className="mt-1 text-sm">{session.message_count}</p>
          </div>
          <div>
            <label className="text-xs text-slate-400 uppercase">Tool Calls</label>
            <p className="mt-1 text-sm">{session.tool_call_count}</p>
          </div>
        </div>
        {session.model && (
          <div>
            <label className="text-xs text-slate-400 uppercase">Model</label>
            <p className="mt-1 text-sm">{session.model}</p>
          </div>
        )}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-xs text-slate-400 uppercase">Created</label>
            <p className="mt-1 text-sm">{new Date(session.created_at).toLocaleDateString()}</p>
          </div>
          {session.last_activity && (
            <div>
              <label className="text-xs text-slate-400 uppercase">Last Activity</label>
              <p className="mt-1 text-sm">{new Date(session.last_activity).toLocaleDateString()}</p>
            </div>
          )}
        </div>
      </div>
    </>
  );
}

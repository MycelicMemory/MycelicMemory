import { useState, useEffect, useRef, useCallback } from 'react';
import toast from 'react-hot-toast';
import { Network, RefreshCw, ZoomIn, ZoomOut, Maximize2, MessageSquare, Layers, Eye, EyeOff } from 'lucide-react';
import type { Memory, MemoryRelationship, ClaudeSession } from '../../shared/types';

// Note: vis-network types
interface NetworkNode {
  id: string;
  label: string;
  title?: string;
  color?: string | {
    background: string;
    border: string;
    highlight?: { background: string; border: string };
    hover?: { background: string; border: string };
  };
  shape?: string;
  size?: number;
  font?: { color?: string; size?: number };
  opacity?: number;
  hidden?: boolean;
}

interface NetworkEdge {
  id: string;
  from: string;
  to: string;
  label?: string;
  title?: string;
  color?: string | { color: string; opacity?: number; hover?: string };
  width?: number;
  arrows?: string | { to?: { enabled: boolean; scaleFactor?: number } };
  dashes?: boolean;
  hoverWidth?: number;
  hidden?: boolean;
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

// Improvement 10: Expanded with actual project domains
const DOMAIN_COLORS: Record<string, string> = {
  general: '#6366f1',
  architecture: '#3b82f6',
  schema: '#a78bfa',
  build: '#f472b6',
  api: '#34d399',
  mcp: '#fbbf24',
  pipeline: '#818cf8',
  recall: '#fb923c',
  relationships: '#e879f9',
  desktop: '#2dd4bf',
  config: '#94a3b8',
  search: '#c084fc',
  operations: '#38bdf8',
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
  const nodesDataSetRef = useRef<any>(null);
  const edgesDataSetRef = useRef<any>(null);
  const [memories, setMemories] = useState<Memory[]>([]);
  const [relationships, setRelationships] = useState<MemoryRelationship[]>([]);
  const [sessions, setSessions] = useState<ClaudeSession[]>([]);
  const [selectedItem, setSelectedItem] = useState<SelectedItem | null>(null);
  const [loading, setLoading] = useState(true);
  const [discovering, setDiscovering] = useState(false);
  const [showSessions, setShowSessions] = useState(true);
  const [showOrphans, setShowOrphans] = useState(false);
  const [clustered, setClustered] = useState(false);
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
  }, [memories, relationships, sessions, filter, showSessions, showOrphans, clustered]);

  async function fetchData() {
    try {
      setLoading(true);
      // Load memories and sessions only - relationships are fetched per-memory or via discover button
      // DO NOT call relationships.discover() here - it uses Ollama AI and takes 200+ seconds
      const [memoriesRes, sessionsRes] = await Promise.all([
        window.mycelicMemory.memory.list({ limit: 200 }),
        window.mycelicMemory.claude.sessions().catch(() => []),
      ]);
      setMemories(memoriesRes || []);
      setSessions(sessionsRes || []);

      // Load all relationships in a single call (much faster than per-memory fetching)
      try {
        const allRels = await window.mycelicMemory.relationships.getAll({ limit: 1000 });
        if (Array.isArray(allRels)) {
          setRelationships(allRels);
        }
      } catch {
        // Fallback: if getAll not available, try per-memory approach
        if (memoriesRes && memoriesRes.length > 0) {
          const allRelationships: MemoryRelationship[] = [];
          const seen = new Set<string>();
          const toCheck = memoriesRes.slice(0, 50);
          const relResults = await Promise.allSettled(
            toCheck.map((m: Memory) =>
              window.mycelicMemory.relationships.get(m.id).catch(() => [])
            )
          );
          for (const result of relResults) {
            if (result.status === 'fulfilled' && Array.isArray(result.value)) {
              for (const rel of result.value) {
                if (!seen.has(rel.id)) {
                  seen.add(rel.id);
                  allRelationships.push(rel);
                }
              }
            }
          }
          setRelationships(allRelationships);
        }
      }
    } catch (err) {
      console.error('Failed to fetch data:', err);
    } finally {
      setLoading(false);
    }
  }

  async function handleDiscoverRelationships() {
    try {
      setDiscovering(true);
      // Discover runs keyword-based discovery on the backend, then returns all relationships
      const allRels = await window.mycelicMemory.relationships.discover();
      if (Array.isArray(allRels)) {
        setRelationships(allRels);
        toast.success(`Discovery complete — ${allRels.length} connections`);
      }
    } catch (err) {
      console.error('Failed to discover relationships:', err);
      toast.error('Failed to discover relationships');
    } finally {
      setDiscovering(false);
    }
  }

  const initializeNetwork = useCallback(async () => {
    if (!containerRef.current) return;

    const vis = await import('vis-network/standalone');

    // Filter memories
    let filteredMemories = [...memories];
    let filteredRelationships = relationships;

    if (filter.domain) {
      filteredMemories = memories.filter((m) => m.domain === filter.domain);
      const memoryIds = new Set(filteredMemories.map((m) => m.id));
      filteredRelationships = relationships.filter(
        (r) => memoryIds.has(r.source_memory_id) && memoryIds.has(r.target_memory_id)
      );
    }

    if (filter.relationshipType) {
      filteredRelationships = filteredRelationships.filter(
        (r) => r.relationship_type === filter.relationshipType
      );
    }

    // Improvement 4: Pre-compute edge degree per node
    const degreeMap = new Map<string, number>();
    filteredRelationships.forEach((rel) => {
      degreeMap.set(rel.source_memory_id, (degreeMap.get(rel.source_memory_id) || 0) + 1);
      degreeMap.set(rel.target_memory_id, (degreeMap.get(rel.target_memory_id) || 0) + 1);
    });

    // Improvement 9: Orphan filtering
    if (!showOrphans) {
      const connectedNodeIds = new Set<string>();
      filteredRelationships.forEach((rel) => {
        connectedNodeIds.add(rel.source_memory_id);
        connectedNodeIds.add(rel.target_memory_id);
      });
      // Session links also count as connections
      if (showSessions) {
        filteredMemories.forEach((m) => {
          if (m.conversation_id) connectedNodeIds.add(m.id);
        });
      }
      filteredMemories = filteredMemories.filter((m) => connectedNodeIds.has(m.id));
    }

    // Build memory nodes
    const nodes: NetworkNode[] = filteredMemories.map((memory) => {
      const isSessionSummary = memory.source === 'claude-code-session';
      const degree = degreeMap.get(memory.id) || 0;

      // Improvement 4: Combined score from importance + degree
      const combinedScore = (memory.importance / 10) * 0.6 + (Math.min(degree, 15) / 15) * 0.4;
      const nodeSize = isSessionSummary ? 18 : 8 + combinedScore * 32;

      const domainColor = DOMAIN_COLORS[memory.domain || 'general'] || DOMAIN_COLORS.general;
      const bgColor = isSessionSummary ? SESSION_NODE_COLOR : domainColor;

      return {
        id: memory.id,
        // Improvement 5: Shorter labels (20 chars)
        label: memory.content.substring(0, 20) + (memory.content.length > 20 ? '...' : ''),
        title: memory.content,
        color: {
          background: bgColor,
          border: isSessionSummary ? '#ea580c' : '#1e293b',
          // Improvement 8: Hover/selection styling
          highlight: { background: bgColor, border: '#f1f5f9' },
          hover: { background: bgColor, border: '#60a5fa' },
        },
        shape: isSessionSummary ? 'diamond' : 'dot',
        size: nodeSize,
      };
    });

    // Build session nodes (if enabled)
    const sessionNodeMap = new Map<string, ClaudeSession>();
    if (showSessions) {
      // Find which sessions are referenced by memories
      const referencedSessionIds = new Set(
        filteredMemories
          .filter((m) => m.conversation_id)
          .map((m) => m.conversation_id!)
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
              highlight: { background: projectColor, border: '#f1f5f9' },
              hover: { background: projectColor, border: '#60a5fa' },
            },
            shape: 'star',
            size: 20,
            font: { color: '#f1f5f9', size: 11 },
          });
        }
      });
    }

    // Improvement 6: Edge styling — reduce visual noise
    const edges: NetworkEdge[] = filteredRelationships.map((rel) => {
      const edgeColor = RELATIONSHIP_COLORS[rel.relationship_type] || '#6366f1';
      const opacity = 0.5 + (rel.strength || 0.5) * 0.5;
      return {
        id: rel.id,
        from: rel.source_memory_id,
        to: rel.target_memory_id,
        title: rel.relationship_type,
        color: { color: edgeColor, opacity, hover: edgeColor },
        width: 0.8,
        hoverWidth: 2,
        arrows: { to: { enabled: true, scaleFactor: 0.5 } },
      };
    });

    // Build session-to-memory edges (dashed orange)
    if (showSessions) {
      filteredMemories.forEach((memory) => {
        if (memory.conversation_id && sessionNodeMap.has(memory.conversation_id)) {
          edges.push({
            id: `trace:${memory.id}`,
            from: memory.id,
            to: `session:${memory.conversation_id}`,
            color: { color: SOURCE_EDGE_COLOR, opacity: 0.5 },
            width: 1,
            arrows: { to: { enabled: true, scaleFactor: 0.4 } },
            dashes: true,
          });
        }
      });
    }

    const nodesDataSet = new vis.DataSet(nodes);
    const edgesDataSet = new vis.DataSet(edges);
    nodesDataSetRef.current = nodesDataSet;
    edgesDataSetRef.current = edgesDataSet;

    const data = { nodes: nodesDataSet, edges: edgesDataSet };

    const options = {
      nodes: {
        font: {
          color: '#f1f5f9',
          size: 12,
        },
        borderWidth: 2,
        // Improvement 8: Selected border width
        borderWidthSelected: 4,
        // Improvement 5: Dynamic label visibility
        scaling: {
          label: {
            enabled: true,
            min: 8,
            max: 20,
            drawThreshold: 8,
            maxVisible: 20,
          },
        },
      },
      edges: {
        font: {
          color: '#94a3b8',
          size: 10,
          strokeWidth: 0,
        },
        // Improvement 6: Dynamic smooth type
        smooth: {
          type: 'dynamic',
        },
      },
      // Improvement 1: Physics engine overhaul — forceAtlas2Based
      physics: {
        enabled: true,
        forceAtlas2Based: {
          gravitationalConstant: -80,
          centralGravity: 0.005,
          springLength: 230,
          springConstant: 0.08,
          damping: 0.4,
          avoidOverlap: 0.8,
        },
        stabilization: {
          enabled: true,
          iterations: 300,
          updateInterval: 25,
          fit: true,
        },
        maxVelocity: 30,
        minVelocity: 0.1,
        timestep: 0.35,
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

    // Improvement 2: Stabilize-then-freeze — stop physics after layout settles
    networkRef.current.on('stabilizationIterationsDone', () => {
      networkRef.current?.setOptions({ physics: { enabled: false } });
      networkRef.current?.fit();
    });

    networkRef.current.on('dragStart', () => {
      networkRef.current?.setOptions({ physics: { enabled: true } });
    });

    networkRef.current.on('dragEnd', () => {
      setTimeout(() => {
        networkRef.current?.setOptions({ physics: { enabled: false } });
      }, 1500);
    });

    // Improvement 3: Neighbourhood highlighting on hover
    networkRef.current.on('hoverNode', (params: { node: string }) => {
      const hoveredId = params.node;
      const connectedNodes = new Set<string>(
        networkRef.current.getConnectedNodes(hoveredId) as string[]
      );
      connectedNodes.add(hoveredId);
      const connectedEdgeIds = new Set<string>(
        networkRef.current.getConnectedEdges(hoveredId) as string[]
      );

      const nodeUpdates: { id: string; opacity: number }[] = [];
      nodesDataSet.forEach((node: NetworkNode) => {
        if (!connectedNodes.has(node.id)) {
          nodeUpdates.push({ id: node.id, opacity: 0.15 });
        }
      });
      if (nodeUpdates.length > 0) nodesDataSet.update(nodeUpdates);

      const edgeUpdates: { id: string; hidden: boolean }[] = [];
      edgesDataSet.forEach((edge: NetworkEdge) => {
        if (!connectedEdgeIds.has(edge.id)) {
          edgeUpdates.push({ id: edge.id, hidden: true });
        }
      });
      if (edgeUpdates.length > 0) edgesDataSet.update(edgeUpdates);
    });

    networkRef.current.on('blurNode', () => {
      const nodeUpdates: { id: string; opacity: number }[] = [];
      nodesDataSet.forEach((node: NetworkNode) => {
        nodeUpdates.push({ id: node.id, opacity: 1.0 });
      });
      if (nodeUpdates.length > 0) nodesDataSet.update(nodeUpdates);

      const edgeUpdates: { id: string; hidden: boolean }[] = [];
      edgesDataSet.forEach((edge: NetworkEdge) => {
        edgeUpdates.push({ id: edge.id, hidden: false });
      });
      if (edgeUpdates.length > 0) edgesDataSet.update(edgeUpdates);
    });

    // Selection handlers
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

    // Improvement 7: Apply domain clustering if enabled
    if (clustered) {
      const domainSet = new Set(filteredMemories.map((m) => m.domain || 'general'));
      domainSet.forEach((domain) => {
        const domainMemoryIds = new Set(
          filteredMemories
            .filter((m) => (m.domain || 'general') === domain)
            .map((m) => m.id)
        );
        const count = domainMemoryIds.size;
        if (count < 2) return;

        networkRef.current.cluster({
          joinCondition: (nodeOptions: { id: string }) => domainMemoryIds.has(nodeOptions.id),
          clusterNodeProperties: {
            id: `cluster:${domain}`,
            label: `${domain} (${count})`,
            color: {
              background: DOMAIN_COLORS[domain] || DOMAIN_COLORS.general,
              border: '#f1f5f9',
            },
            shape: 'dot',
            size: 35,
            font: { color: '#f1f5f9', size: 14 },
            borderWidth: 3,
          },
        });
      });

      networkRef.current.on('doubleClick', (params: { nodes: string[] }) => {
        if (params.nodes.length === 1 && networkRef.current.isCluster(params.nodes[0])) {
          networkRef.current.openCluster(params.nodes[0]);
        }
      });
    }
  }, [memories, relationships, sessions, filter, showSessions, showOrphans, clustered]);

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
  const linkedCount = memories.filter((m) => m.conversation_id).length;
  const sessionNodeCount = showSessions
    ? new Set(memories.filter((m) => m.conversation_id).map((m) => m.conversation_id)).size
    : 0;

  // Compute orphan count for toolbar display
  const orphanNodeIds = new Set<string>();
  const connectedIds = new Set<string>();
  relationships.forEach((rel) => {
    connectedIds.add(rel.source_memory_id);
    connectedIds.add(rel.target_memory_id);
  });
  if (showSessions) {
    memories.forEach((m) => {
      if (m.conversation_id) connectedIds.add(m.id);
    });
  }
  memories.forEach((m) => {
    if (!connectedIds.has(m.id)) orphanNodeIds.add(m.id);
  });
  const orphanCount = orphanNodeIds.size;

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

          {/* Improvement 7: Cluster Toggle */}
          <button
            onClick={() => setClustered(!clustered)}
            className={`px-3 py-1.5 rounded-lg text-sm flex items-center gap-2 transition-colors ${
              clustered
                ? 'bg-blue-500/20 text-blue-400 border border-blue-500/50'
                : 'bg-slate-800 text-slate-400 border border-slate-700'
            }`}
            title="Cluster nodes by domain"
          >
            <Layers className="w-4 h-4" />
            Cluster
          </button>

          {/* Improvement 9: Orphan Toggle */}
          <button
            onClick={() => setShowOrphans(!showOrphans)}
            className={`px-3 py-1.5 rounded-lg text-sm flex items-center gap-2 transition-colors ${
              showOrphans
                ? 'bg-purple-500/20 text-purple-400 border border-purple-500/50'
                : 'bg-slate-800 text-slate-400 border border-slate-700'
            }`}
            title={showOrphans ? 'Hide orphan nodes' : 'Show orphan nodes'}
          >
            {showOrphans ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
            Orphans{orphanCount > 0 ? ` (${orphanCount})` : ''}
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

      {/* Improvement 10: Enhanced legend with actual domain colors */}
      <div className="p-3 border-t border-slate-700 flex items-center gap-4 flex-wrap">
        <span className="text-xs text-slate-500">Domains:</span>
        {domains.map((domain) => (
          <button
            key={domain}
            onClick={() =>
              setFilter({ ...filter, domain: filter.domain === domain ? '' : domain })
            }
            className={`flex items-center gap-1.5 transition-opacity ${
              filter.domain && filter.domain !== domain ? 'opacity-40' : ''
            }`}
          >
            <div
              className="w-2.5 h-2.5 rounded-full"
              style={{ backgroundColor: DOMAIN_COLORS[domain] || DOMAIN_COLORS.general }}
            />
            <span className="text-xs text-slate-400 hover:text-slate-200">{domain}</span>
          </button>
        ))}
        <div className="h-4 w-px bg-slate-700" />
        <span className="text-xs text-slate-500">Edges:</span>
        {relationshipTypes.map((type) => (
          <button
            key={type}
            onClick={() =>
              setFilter({
                ...filter,
                relationshipType: filter.relationshipType === type ? '' : type,
              })
            }
            className={`flex items-center gap-1.5 transition-opacity ${
              filter.relationshipType && filter.relationshipType !== type ? 'opacity-40' : ''
            }`}
          >
            <div
              className="w-4 h-0.5"
              style={{ backgroundColor: RELATIONSHIP_COLORS[type] || '#6366f1' }}
            />
            <span className="text-xs text-slate-400 hover:text-slate-200">{type}</span>
          </button>
        ))}
        <div className="h-4 w-px bg-slate-700" />
        <div className="flex items-center gap-1.5">
          <svg width="12" height="12" viewBox="0 0 14 14">
            <polygon
              points="7,1 13,7 7,13 1,7"
              fill={SESSION_NODE_COLOR}
              stroke="#1e293b"
              strokeWidth="1"
            />
          </svg>
          <span className="text-xs text-slate-400">summary</span>
        </div>
        <div className="flex items-center gap-1.5">
          <svg width="12" height="12" viewBox="0 0 14 14">
            <polygon
              points="7,0 9,5 14,5 10,8 12,14 7,10 2,14 4,8 0,5 5,5"
              fill={SESSION_NODE_COLOR}
              stroke="#1e293b"
              strokeWidth="0.5"
            />
          </svg>
          <span className="text-xs text-slate-400">chat</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div
            className="w-5 border-t border-dashed"
            style={{ borderColor: SOURCE_EDGE_COLOR }}
          />
          <span className="text-xs text-slate-400">traced</span>
        </div>
      </div>
    </div>
  );
}

function MemoryDetailPanel({ memory, sessions }: { memory: Memory; sessions: ClaudeSession[] }) {
  const linkedSession = memory.conversation_id
    ? sessions.find((s) => s.id === memory.conversation_id)
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

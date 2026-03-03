import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import toast from 'react-hot-toast';
import { Network, RefreshCw, ZoomIn, ZoomOut, Maximize2, MessageSquare, Layers, Eye, EyeOff, Search, X, Settings2 } from 'lucide-react';
import type { Memory, MemoryRelationship, ClaudeSession, GraphView, GraphFilterState } from '../../shared/types';
import GraphContextMenu, { type ContextMenuAction } from '../components/graph/GraphContextMenu';
import GraphDisplaySettings from '../components/graph/GraphDisplaySettings';
import GraphViewManager from '../components/graph/GraphViewManager';
import { useGraphSettings, DEFAULT_PHYSICS, DEFAULT_STYLE } from '../hooks/useGraphSettings';
import { useGraphViews } from '../hooks/useGraphViews';

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

const SESSION_COLORS = [
  '#f97316', '#06b6d4', '#a855f7', '#ec4899',
  '#84cc16', '#eab308', '#14b8a6', '#f43f5e',
];

const SESSION_NODE_COLOR = '#f97316';
const SOURCE_EDGE_COLOR = '#f97316';

function buildNetworkOptions(physics: typeof DEFAULT_PHYSICS, style: typeof DEFAULT_STYLE) {
  return {
    nodes: {
      font: { color: '#f1f5f9', size: style.nodeFontSize },
      borderWidth: style.nodeBorderWidth,
      borderWidthSelected: 4,
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
      font: { color: '#94a3b8', size: style.edgeFontSize, strokeWidth: 0 },
      smooth: { type: style.edgeSmoothType as any },
    },
    physics: {
      enabled: true,
      forceAtlas2Based: {
        gravitationalConstant: physics.gravitationalConstant,
        centralGravity: physics.centralGravity,
        springLength: physics.springLength,
        springConstant: physics.springConstant,
        damping: physics.damping,
        avoidOverlap: physics.avoidOverlap,
      },
      stabilization: {
        enabled: true,
        iterations: 300,
        updateInterval: 25,
        fit: true,
      },
      maxVelocity: physics.maxVelocity,
      minVelocity: 0.1,
      timestep: physics.timestep,
    },
    interaction: {
      hover: true,
      tooltipDelay: 200,
      zoomView: true,
      dragView: true,
    },
  };
}

type SelectedItem =
  | { type: 'memory'; data: Memory }
  | { type: 'session'; data: ClaudeSession };

export default function KnowledgeGraph() {
  const containerRef = useRef<HTMLDivElement>(null);
  const networkRef = useRef<any>(null);
  const nodesDataSetRef = useRef<any>(null);
  const edgesDataSetRef = useRef<any>(null);
  const visModuleRef = useRef<any>(null);

  // Fix 3: Refs for tracking dimmed/hidden state (smart hover diffing)
  const dimmedNodesRef = useRef<Set<string>>(new Set());
  const hiddenEdgesRef = useRef<Set<string>>(new Set());

  // Fix 2: Refs for stable event handler closures (no stacking)
  const memoriesRef = useRef<Memory[]>([]);
  const sessionNodeMapRef = useRef<Map<string, ClaudeSession>>(new Map());

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
  const [searchQuery, setSearchQuery] = useState('');
  const [renderedCounts, setRenderedCounts] = useState({ nodes: 0, edges: 0 });

  // New state for context menu, hidden/pinned nodes, display settings
  const [contextMenu, setContextMenu] = useState<{
    x: number; y: number; nodeId: string | null; nodeType: 'memory' | 'session' | null;
  } | null>(null);
  const [hiddenNodeIds, setHiddenNodeIds] = useState<Set<string>>(new Set());
  const [pinnedMemoryIds, setPinnedMemoryIds] = useState<Set<string>>(new Set());
  const [showDisplaySettings, setShowDisplaySettings] = useState(false);

  // Hooks
  const { physics, style, tuning, setTuning, tuningRef, applyPhysics, applyStyle, resetToDefaults } = useGraphSettings(networkRef);
  const { views, activeViewId, saveView, deleteView } = useGraphViews();

  // Keep refs in sync so event handlers always see current data
  useEffect(() => { memoriesRef.current = memories; }, [memories]);

  // Ref for hiddenNodeIds so context menu handler always sees current value
  const hiddenNodeIdsRef = useRef(hiddenNodeIds);
  useEffect(() => { hiddenNodeIdsRef.current = hiddenNodeIds; }, [hiddenNodeIds]);

  // Memoize render-phase computations (previously recomputed every render)
  const domains = useMemo(
    () => [...new Set(memories.map((m) => m.domain || 'general'))],
    [memories]
  );
  const relationshipTypes = useMemo(
    () => [...new Set(relationships.map((r) => r.relationship_type))],
    [relationships]
  );
  const { orphanCount, linkedCount, sessionNodeCount } = useMemo(() => {
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
    return {
      orphanCount: memories.filter((m) => !connectedIds.has(m.id)).length,
      linkedCount: memories.filter((m) => m.conversation_id).length,
      sessionNodeCount: showSessions
        ? new Set(memories.filter((m) => m.conversation_id).map((m) => m.conversation_id)).size
        : 0,
    };
  }, [memories, relationships, showSessions]);

  useEffect(() => {
    fetchData();
  }, []);

  async function fetchData() {
    try {
      setLoading(true);
      const [memoriesRes, sessionsRes] = await Promise.all([
        window.mycelicMemory.memory.list({ limit: 200 }),
        window.mycelicMemory.claude.sessions().catch(() => []),
      ]);
      setMemories(memoriesRes || []);
      setSessions(sessionsRes || []);

      try {
        const allRels = await window.mycelicMemory.relationships.getAll({ limit: 1000 });
        if (Array.isArray(allRels)) {
          setRelationships(allRels);
        }
      } catch {
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

  function handleSearch(query: string) {
    setSearchQuery(query);
    if (!networkRef.current || !nodesDataSetRef.current) return;

    const nodesDS = nodesDataSetRef.current;

    if (!query.trim()) {
      // Clear search — restore all nodes to full opacity
      const updates: { id: string; opacity: number }[] = [];
      nodesDS.forEach((node: NetworkNode) => {
        updates.push({ id: node.id, opacity: 1.0 });
      });
      if (updates.length > 0) nodesDS.update(updates);
      return;
    }

    const lowerQuery = query.toLowerCase();
    const matchingIds = new Set<string>();
    memoriesRef.current.forEach((m) => {
      if (
        m.content.toLowerCase().includes(lowerQuery) ||
        (m.domain || '').toLowerCase().includes(lowerQuery) ||
        (m.tags || []).some((t) => t.toLowerCase().includes(lowerQuery))
      ) {
        matchingIds.add(m.id);
      }
    });

    // Dim non-matching, highlight matching
    const updates: { id: string; opacity: number }[] = [];
    nodesDS.forEach((node: NetworkNode) => {
      updates.push({
        id: node.id,
        opacity: matchingIds.has(node.id) ? 1.0 : 0.15,
      });
    });
    if (updates.length > 0) nodesDS.update(updates);

    // Focus camera on first match
    if (matchingIds.size > 0 && matchingIds.size <= 20) {
      networkRef.current.fit({ nodes: Array.from(matchingIds), animation: true });
    }
  }

  // Pure computation: build nodes + edges from current state
  const buildGraphData = useCallback(() => {
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

    // Pre-compute edge degree per node
    const degreeMap = new Map<string, number>();
    filteredRelationships.forEach((rel) => {
      degreeMap.set(rel.source_memory_id, (degreeMap.get(rel.source_memory_id) || 0) + 1);
      degreeMap.set(rel.target_memory_id, (degreeMap.get(rel.target_memory_id) || 0) + 1);
    });

    // Orphan filtering
    if (!showOrphans) {
      const connectedNodeIds = new Set<string>();
      filteredRelationships.forEach((rel) => {
        connectedNodeIds.add(rel.source_memory_id);
        connectedNodeIds.add(rel.target_memory_id);
      });
      if (showSessions) {
        filteredMemories.forEach((m) => {
          if (m.conversation_id) connectedNodeIds.add(m.id);
        });
      }
      filteredMemories = filteredMemories.filter((m) => connectedNodeIds.has(m.id));
    }

    // Hidden nodes filter
    if (hiddenNodeIds.size > 0) {
      filteredMemories = filteredMemories.filter((m) => !hiddenNodeIds.has(m.id));
      filteredRelationships = filteredRelationships.filter(
        (r) => !hiddenNodeIds.has(r.source_memory_id) && !hiddenNodeIds.has(r.target_memory_id)
      );
    }

    // Build memory nodes
    const nodes: NetworkNode[] = filteredMemories.map((memory) => {
      const isSessionSummary = memory.source === 'claude-code-session';
      const degree = degreeMap.get(memory.id) || 0;
      const combinedScore = (memory.importance / 10) * 0.6 + (Math.min(degree, 15) / 15) * 0.4;
      const nodeSize = isSessionSummary ? 18 : 8 + combinedScore * 32;
      const domainColor = DOMAIN_COLORS[memory.domain || 'general'] || DOMAIN_COLORS.general;
      const bgColor = isSessionSummary ? SESSION_NODE_COLOR : domainColor;

      const contentPreview = memory.content.length > 120
        ? memory.content.substring(0, 120) + '...'
        : memory.content;
      const tooltip = [
        `[${memory.domain || 'general'}] importance: ${memory.importance}/10`,
        `connections: ${degree}`,
        '',
        contentPreview,
      ].join('\n');

      return {
        id: memory.id,
        label: memory.content.substring(0, 20) + (memory.content.length > 20 ? '...' : ''),
        title: tooltip,
        color: {
          background: bgColor,
          border: isSessionSummary ? '#ea580c' : '#1e293b',
          highlight: { background: bgColor, border: '#f1f5f9' },
          hover: { background: bgColor, border: '#60a5fa' },
        },
        shape: isSessionSummary ? 'diamond' : 'dot',
        size: nodeSize,
      };
    });

    // Build session nodes
    const sessionNodeMap = new Map<string, ClaudeSession>();
    if (showSessions) {
      const referencedSessionIds = new Set(
        filteredMemories.filter((m) => m.conversation_id).map((m) => m.conversation_id!)
      );
      const projectColors = new Map<string, string>();
      let colorIdx = 0;

      sessions.forEach((s) => {
        if (referencedSessionIds.has(s.id) && !hiddenNodeIds.has(`session:${s.id}`)) {
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

    // Build relationship edges
    const edges: NetworkEdge[] = filteredRelationships.map((rel) => {
      const edgeColor = RELATIONSHIP_COLORS[rel.relationship_type] || '#6366f1';
      const opacity = 0.5 + (rel.strength || 0.5) * 0.5;
      return {
        id: rel.id,
        from: rel.source_memory_id,
        to: rel.target_memory_id,
        title: `${rel.relationship_type} (strength: ${(rel.strength || 0.5).toFixed(2)})`,
        color: { color: edgeColor, opacity, hover: edgeColor },
        width: 0.5 + (rel.strength || 0.5) * 2.5,
        hoverWidth: 3,
        arrows: { to: { enabled: true, scaleFactor: 0.5 } },
      };
    });

    // Build session-to-memory edges
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

    return { nodes, edges, sessionNodeMap, filteredMemories };
  }, [memories, relationships, sessions, filter, showSessions, showOrphans, hiddenNodeIds]);

  // Fix 2: Register event handlers ONCE — uses refs for current data, not closures over stale state
  const registerEventHandlers = useCallback((network: any) => {
    // Stabilize-then-freeze (skip freeze while user is tuning physics sliders)
    network.on('stabilizationIterationsDone', () => {
      if (tuningRef.current) return;
      network.setOptions({ physics: { enabled: false } });
      network.fit();
    });

    network.on('dragStart', () => {
      network.setOptions({ physics: { enabled: true } });
    });

    network.on('dragEnd', () => {
      setTimeout(() => {
        if (!tuningRef.current) {
          network.setOptions({ physics: { enabled: false } });
        }
      }, 1500);
    });

    // Context menu on right-click
    network.on('oncontext', (params: { event: MouseEvent; pointer: { DOM: { x: number; y: number } } }) => {
      params.event.preventDefault();
      const nodeId = network.getNodeAt(params.pointer.DOM) as string | undefined;
      let nodeType: 'memory' | 'session' | null = null;
      let resolvedNodeId: string | null = nodeId ?? null;

      if (nodeId) {
        nodeType = nodeId.startsWith('session:') ? 'session' : 'memory';
      }

      // Use the container's bounding rect to get viewport-relative coords
      const container = network.body.container as HTMLElement;
      const rect = container.getBoundingClientRect();

      setContextMenu({
        x: rect.left + params.pointer.DOM.x,
        y: rect.top + params.pointer.DOM.y,
        nodeId: resolvedNodeId,
        nodeType,
      });
    });

    // Fix 3: Smart hover diffing — only update the delta between previous and current hover state
    network.on('hoverNode', (params: { node: string }) => {
      const nodesDS = nodesDataSetRef.current;
      const edgesDS = edgesDataSetRef.current;
      if (!nodesDS || !edgesDS) return;

      const hoveredId = params.node;
      const connectedNodes = new Set<string>(
        network.getConnectedNodes(hoveredId) as string[]
      );
      connectedNodes.add(hoveredId);
      const connectedEdges = new Set<string>(
        network.getConnectedEdges(hoveredId) as string[]
      );

      // Diff nodes: only update items whose dim state changed
      const newDimmed = new Set<string>();
      const nodeUpdates: { id: string; opacity: number }[] = [];

      nodesDS.forEach((node: NetworkNode) => {
        const shouldDim = !connectedNodes.has(node.id);
        const wasDimmed = dimmedNodesRef.current.has(node.id);
        if (shouldDim) {
          newDimmed.add(node.id);
          if (!wasDimmed) nodeUpdates.push({ id: node.id, opacity: 0.15 });
        } else if (wasDimmed) {
          nodeUpdates.push({ id: node.id, opacity: 1.0 });
        }
      });
      dimmedNodesRef.current = newDimmed;
      if (nodeUpdates.length > 0) nodesDS.update(nodeUpdates);

      // Diff edges: only update items whose hidden state changed
      const newHidden = new Set<string>();
      const edgeUpdates: { id: string; hidden: boolean }[] = [];

      edgesDS.forEach((edge: NetworkEdge) => {
        const shouldHide = !connectedEdges.has(edge.id);
        const wasHidden = hiddenEdgesRef.current.has(edge.id);
        if (shouldHide) {
          newHidden.add(edge.id);
          if (!wasHidden) edgeUpdates.push({ id: edge.id, hidden: true });
        } else if (wasHidden) {
          edgeUpdates.push({ id: edge.id, hidden: false });
        }
      });
      hiddenEdgesRef.current = newHidden;
      if (edgeUpdates.length > 0) edgesDS.update(edgeUpdates);
    });

    network.on('blurNode', () => {
      const nodesDS = nodesDataSetRef.current;
      const edgesDS = edgesDataSetRef.current;
      if (!nodesDS || !edgesDS) return;

      // Only restore previously dimmed/hidden items — no full iteration
      if (dimmedNodesRef.current.size > 0) {
        nodesDS.update(
          Array.from(dimmedNodesRef.current).map((id) => ({ id, opacity: 1.0 }))
        );
        dimmedNodesRef.current.clear();
      }
      if (hiddenEdgesRef.current.size > 0) {
        edgesDS.update(
          Array.from(hiddenEdgesRef.current).map((id) => ({ id, hidden: false }))
        );
        hiddenEdgesRef.current.clear();
      }
    });

    // Selection handlers use refs for always-current data
    network.on('selectNode', (params: { nodes: string[] }) => {
      const nodeId = params.nodes[0];
      if (nodeId.startsWith('session:')) {
        const sessId = nodeId.replace('session:', '');
        const session = sessionNodeMapRef.current.get(sessId);
        if (session) {
          setSelectedItem({ type: 'session', data: session });
        }
        return;
      }
      const memory = memoriesRef.current.find((m) => m.id === nodeId);
      if (memory) {
        setSelectedItem({ type: 'memory', data: memory });
      }
    });

    network.on('deselectNode', () => {
      setSelectedItem(null);
    });

    // Cluster double-click — safe to always register, does nothing when no clusters exist
    network.on('doubleClick', (params: { nodes: string[] }) => {
      if (params.nodes.length === 1 && network.isCluster(params.nodes[0])) {
        network.openCluster(params.nodes[0]);
      }
    });
  }, []);

  // Fix 1: Main effect — create network ONCE, update DataSets on subsequent changes
  useEffect(() => {
    if (memories.length === 0 || !containerRef.current) return;

    let cancelled = false;

    (async () => {
      // Cache the vis-network import across renders
      if (!visModuleRef.current) {
        visModuleRef.current = await import('vis-network/standalone');
      }
      if (cancelled || !containerRef.current) return;

      const vis = visModuleRef.current;
      const { nodes, edges, sessionNodeMap, filteredMemories } = buildGraphData();
      sessionNodeMapRef.current = sessionNodeMap;
      setRenderedCounts({ nodes: nodes.length, edges: edges.length });

      if (!networkRef.current) {
        // FIRST TIME: create network + DataSets + register all event handlers once
        const nodesDS = new vis.DataSet(nodes);
        const edgesDS = new vis.DataSet(edges);
        nodesDataSetRef.current = nodesDS;
        edgesDataSetRef.current = edgesDS;

        const initialOptions = buildNetworkOptions(physics, style);
        networkRef.current = new vis.Network(
          containerRef.current!,
          { nodes: nodesDS, edges: edgesDS },
          initialOptions
        );
        registerEventHandlers(networkRef.current);
      } else {
        // SUBSEQUENT: incremental DataSet update — NO destroy/recreate
        dimmedNodesRef.current.clear();
        hiddenEdgesRef.current.clear();
        nodesDataSetRef.current.clear();
        nodesDataSetRef.current.add(nodes);
        edgesDataSetRef.current.clear();
        edgesDataSetRef.current.add(edges);
      }

      // Apply clustering if enabled
      if (clustered && networkRef.current) {
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
      }

      // Re-stabilize layout (brief physics burst, then freeze)
      if (networkRef.current) {
        networkRef.current.setOptions({ physics: { enabled: true } });
        networkRef.current.stabilize(150);
      }
    })();

    return () => { cancelled = true; };
  }, [memories, relationships, sessions, filter, showSessions, showOrphans, clustered, hiddenNodeIds, buildGraphData, registerEventHandlers]);

  // Cleanup on unmount only — network is never destroyed mid-lifecycle
  useEffect(() => {
    return () => {
      if (networkRef.current) {
        networkRef.current.destroy();
        networkRef.current = null;
      }
    };
  }, []);

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

  // Context menu action handler
  function handleContextMenuAction(action: ContextMenuAction) {
    if (!contextMenu) return;
    const { nodeId } = contextMenu;

    switch (action) {
      case 'focus-neighborhood': {
        if (!nodeId || !networkRef.current) break;
        const connected = networkRef.current.getConnectedNodes(nodeId) as string[];
        networkRef.current.fit({ nodes: [nodeId, ...connected], animation: true });
        break;
      }
      case 'show-only-connected': {
        if (!nodeId || !networkRef.current) break;
        const connected = new Set(networkRef.current.getConnectedNodes(nodeId) as string[]);
        connected.add(nodeId);
        // Hide everything not connected
        const toHide = new Set(hiddenNodeIds);
        memoriesRef.current.forEach((m) => {
          if (!connected.has(m.id)) toHide.add(m.id);
        });
        setHiddenNodeIds(toHide);
        break;
      }
      case 'hide-node': {
        if (!nodeId) break;
        setHiddenNodeIds((prev) => new Set(prev).add(nodeId));
        toast.success('Node hidden');
        break;
      }
      case 'pin-to-view': {
        if (!nodeId) break;
        setPinnedMemoryIds((prev) => new Set(prev).add(nodeId));
        toast.success('Node pinned to view');
        break;
      }
      case 'unpin-from-view': {
        if (!nodeId) break;
        setPinnedMemoryIds((prev) => {
          const next = new Set(prev);
          next.delete(nodeId);
          return next;
        });
        toast.success('Node unpinned');
        break;
      }
      case 'copy-content': {
        if (!nodeId) break;
        const mem = memoriesRef.current.find((m) => m.id === nodeId);
        if (mem) {
          navigator.clipboard.writeText(mem.content);
          toast.success('Content copied');
        }
        break;
      }
      case 'copy-id': {
        if (!nodeId) break;
        const rawId = nodeId.startsWith('session:') ? nodeId.replace('session:', '') : nodeId;
        navigator.clipboard.writeText(rawId);
        toast.success('ID copied');
        break;
      }
      case 'show-all-hidden': {
        setHiddenNodeIds(new Set());
        toast.success('All hidden nodes restored');
        break;
      }
      case 'reset-view': {
        setHiddenNodeIds(new Set());
        setPinnedMemoryIds(new Set());
        setFilter({ domain: '', relationshipType: '' });
        setSearchQuery('');
        handleSearch('');
        resetToDefaults();
        toast.success('View reset');
        break;
      }
      case 'fit-all': {
        handleFit();
        break;
      }
    }
    setContextMenu(null);
  }

  // Load a saved view
  function handleLoadView(view: GraphView) {
    // Restore filters
    setFilter({
      domain: view.filter.domain,
      relationshipType: view.filter.relationshipType,
    });
    setSearchQuery(view.filter.searchQuery);
    setShowSessions(view.filter.showSessions);
    setShowOrphans(view.filter.showOrphans);
    setClustered(view.filter.clustered);

    // Restore hidden/pinned
    setHiddenNodeIds(new Set(view.hiddenNodeIds || []));
    setPinnedMemoryIds(new Set(view.pinnedMemoryIds || []));

    // Restore physics/style
    applyPhysics(view.physics);
    applyStyle(view.style);

    // Apply search highlight after a tick (needs nodesDS to be populated)
    if (view.filter.searchQuery) {
      setTimeout(() => handleSearch(view.filter.searchQuery), 100);
    }

    toast.success(`Loaded view: ${view.name}`);
  }

  // Current filter state for view saving
  const currentFilterState: GraphFilterState = {
    domain: filter.domain,
    relationshipType: filter.relationshipType,
    searchQuery,
    showSessions,
    showOrphans,
    clustered,
  };

  return (
    <div className="h-screen flex flex-col bg-slate-900">
      {/* Toolbar */}
      <div className="p-4 border-b border-slate-700 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h1 className="text-xl font-semibold">Knowledge Graph</h1>
          <span className="text-sm text-slate-400">
            {renderedCounts.nodes} nodes &bull; {renderedCounts.edges} edges
            {renderedCounts.nodes !== memories.length && ` (${memories.length} total)`}
            {hiddenNodeIds.size > 0 && ` \u2022 ${hiddenNodeIds.size} hidden`}
          </span>
          <div className="relative">
            <Search className="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-500" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => handleSearch(e.target.value)}
              placeholder="Search nodes..."
              className="pl-8 pr-7 py-1.5 w-44 bg-slate-800 border border-slate-700 rounded-lg text-sm placeholder:text-slate-500 focus:outline-none focus:border-slate-500"
            />
            {searchQuery && (
              <button
                onClick={() => handleSearch('')}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
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

          {/* Cluster Toggle */}
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

          {/* Orphan Toggle */}
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

          {/* Display Settings Toggle */}
          <button
            onClick={() => {
              const next = !showDisplaySettings;
              setShowDisplaySettings(next);
              setTuning(next);
            }}
            className={`p-2 rounded-lg transition-colors ${
              showDisplaySettings
                ? 'bg-indigo-500/20 text-indigo-400'
                : 'hover:bg-slate-700'
            }`}
            title="Display Settings"
          >
            <Settings2 className="w-4 h-4" />
          </button>

          {/* View Manager */}
          <GraphViewManager
            views={views}
            activeViewId={activeViewId}
            currentFilter={currentFilterState}
            currentPhysics={physics}
            currentStyle={style}
            hiddenNodeIds={Array.from(hiddenNodeIds)}
            pinnedMemoryIds={Array.from(pinnedMemoryIds)}
            onLoadView={handleLoadView}
            onSave={saveView}
            onDelete={deleteView}
          />
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

          {/* Display Settings Panel */}
          {showDisplaySettings && (
            <GraphDisplaySettings
              physics={physics}
              style={style}
              onPhysicsChange={applyPhysics}
              onStyleChange={applyStyle}
              onReset={resetToDefaults}
              onClose={() => { setShowDisplaySettings(false); setTuning(false); }}
            />
          )}

          {/* Context Menu */}
          {contextMenu && (
            <GraphContextMenu
              x={contextMenu.x}
              y={contextMenu.y}
              nodeId={contextMenu.nodeId}
              nodeType={contextMenu.nodeType}
              isPinned={contextMenu.nodeId ? pinnedMemoryIds.has(contextMenu.nodeId) : false}
              hiddenCount={hiddenNodeIds.size}
              onAction={handleContextMenuAction}
              onClose={() => setContextMenu(null)}
            />
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

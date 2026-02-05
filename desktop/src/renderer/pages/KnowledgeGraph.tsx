import { useState, useEffect, useRef, useCallback } from 'react';
import { Network, RefreshCw, ZoomIn, ZoomOut, Maximize2, Filter } from 'lucide-react';
import type { Memory, MemoryRelationship } from '../../shared/types';

// Note: vis-network types
interface NetworkNode {
  id: string;
  label: string;
  title?: string;
  color?: string | { background: string; border: string };
  shape?: string;
  size?: number;
}

interface NetworkEdge {
  id: string;
  from: string;
  to: string;
  label?: string;
  color?: string | { color: string };
  width?: number;
  arrows?: string;
}

interface NetworkData {
  nodes: NetworkNode[];
  edges: NetworkEdge[];
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
};

export default function KnowledgeGraph() {
  const containerRef = useRef<HTMLDivElement>(null);
  const networkRef = useRef<any>(null);
  const [memories, setMemories] = useState<Memory[]>([]);
  const [relationships, setRelationships] = useState<MemoryRelationship[]>([]);
  const [selectedMemory, setSelectedMemory] = useState<Memory | null>(null);
  const [loading, setLoading] = useState(true);
  const [discovering, setDiscovering] = useState(false);
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
  }, [memories, relationships, filter]);

  async function fetchData() {
    try {
      setLoading(true);
      const [memoriesRes, relationsRes] = await Promise.all([
        window.mycelicMemory.memory.list({ limit: 100 }),
        window.mycelicMemory.relationships.discover().catch(() => []),
      ]);
      setMemories(memoriesRes || []);
      setRelationships(relationsRes || []);
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

    // Dynamic import of vis-network
    const vis = await import('vis-network/standalone');

    // Filter data based on current filters
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

    // Build nodes
    const nodes: NetworkNode[] = filteredMemories.map((memory) => ({
      id: memory.id,
      label: memory.content.substring(0, 30) + (memory.content.length > 30 ? '...' : ''),
      title: memory.content,
      color: {
        background: DOMAIN_COLORS[memory.domain || 'general'] || DOMAIN_COLORS.general,
        border: '#1e293b',
      },
      shape: 'dot',
      size: 10 + memory.importance * 2,
    }));

    // Build edges
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

    // Destroy previous network if exists
    if (networkRef.current) {
      networkRef.current.destroy();
    }

    networkRef.current = new vis.Network(containerRef.current, data, options);

    // Handle node selection
    networkRef.current.on('selectNode', (params: { nodes: string[] }) => {
      const nodeId = params.nodes[0];
      const memory = memories.find((m) => m.id === nodeId);
      setSelectedMemory(memory || null);
    });

    networkRef.current.on('deselectNode', () => {
      setSelectedMemory(null);
    });
  }, [memories, relationships, filter]);

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

  return (
    <div className="h-screen flex flex-col bg-slate-900">
      {/* Toolbar */}
      <div className="p-4 border-b border-slate-700 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h1 className="text-xl font-semibold">Knowledge Graph</h1>
          <span className="text-sm text-slate-400">
            {memories.length} nodes • {relationships.length} connections
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
            Discover Relationships
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
        {selectedMemory && (
          <div className="w-80 border-l border-slate-700 p-4 bg-slate-800">
            <h3 className="font-semibold mb-4">Memory Details</h3>
            <div className="space-y-4">
              <div>
                <label className="text-xs text-slate-400 uppercase">Content</label>
                <p className="mt-1 text-sm">{selectedMemory.content}</p>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-xs text-slate-400 uppercase">Domain</label>
                  <p className="mt-1 text-sm">{selectedMemory.domain || 'general'}</p>
                </div>
                <div>
                  <label className="text-xs text-slate-400 uppercase">Importance</label>
                  <p className="mt-1 text-sm">{selectedMemory.importance}/10</p>
                </div>
              </div>
              {selectedMemory.tags && selectedMemory.tags.length > 0 && (
                <div>
                  <label className="text-xs text-slate-400 uppercase">Tags</label>
                  <div className="flex flex-wrap gap-1 mt-1">
                    {selectedMemory.tags.map((tag) => (
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
            </div>
          </div>
        )}
      </div>

      {/* Legend */}
      <div className="p-4 border-t border-slate-700 flex items-center gap-6">
        <span className="text-xs text-slate-400">Domains:</span>
        {Object.entries(DOMAIN_COLORS).slice(0, 6).map(([domain, color]) => (
          <div key={domain} className="flex items-center gap-1.5">
            <div className="w-3 h-3 rounded-full" style={{ backgroundColor: color }} />
            <span className="text-xs text-slate-400">{domain}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

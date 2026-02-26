import { useState, useEffect, useCallback } from 'react';
import {
  Database,
  Plus,
  RefreshCw,
  Pause,
  Play,
  Trash2,
  ChevronRight,
  Clock,
  AlertCircle,
  CheckCircle,
  Loader2,
  X,
} from 'lucide-react';
import toast from 'react-hot-toast';
import { ConfirmDialog } from '../components/ConfirmDialog';

interface DataSource {
  id: string;
  source_type: string;
  name: string;
  config: string;
  status: 'active' | 'paused' | 'error';
  last_sync_at?: string;
  last_sync_position?: string;
  created_at: string;
  updated_at: string;
}

interface SyncHistory {
  id: string;
  source_id: string;
  started_at: string;
  completed_at?: string;
  items_processed: number;
  memories_created: number;
  duplicates_skipped: number;
  status: string;
  error?: string;
}

interface SourceStats {
  total_memories: number;
  total_syncs: number;
  successful_syncs: number;
  failed_syncs: number;
  last_sync_at?: string;
  last_error?: string;
}

const SOURCE_TYPES = [
  { value: 'claude-code-local', label: 'Claude Code (Local)', icon: '🤖' },
  { value: 'slack', label: 'Slack', icon: '💬' },
  { value: 'discord', label: 'Discord', icon: '🎮' },
  { value: 'telegram', label: 'Telegram', icon: '📱' },
  { value: 'imessage', label: 'iMessage', icon: '💬' },
  { value: 'email', label: 'Email', icon: '📧' },
  { value: 'browser', label: 'Browser History', icon: '🌐' },
  { value: 'notion', label: 'Notion', icon: '📝' },
  { value: 'obsidian', label: 'Obsidian', icon: '💎' },
  { value: 'github', label: 'GitHub', icon: '🐙' },
  { value: 'custom', label: 'Custom', icon: '⚙️' },
];

const STATUS_COLORS: Record<string, string> = {
  active: 'bg-green-500/20 text-green-400 border-green-500/30',
  paused: 'bg-amber-500/20 text-amber-400 border-amber-500/30',
  error: 'bg-red-500/20 text-red-400 border-red-500/30',
};

const SYNC_STATUS_COLORS: Record<string, string> = {
  running: 'text-blue-400',
  completed: 'text-green-400',
  failed: 'text-red-400',
};

export default function DataSources() {
  const [sources, setSources] = useState<DataSource[]>([]);
  const [selectedSource, setSelectedSource] = useState<DataSource | null>(null);
  const [sourceStats, setSourceStats] = useState<SourceStats | null>(null);
  const [syncHistory, setSyncHistory] = useState<SyncHistory[]>([]);
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState<string | null>(null);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<DataSource | null>(null);

  // Add source form state
  const [newSourceType, setNewSourceType] = useState('claude-code-local');
  const [newSourceName, setNewSourceName] = useState('');
  const [newSourceConfig, setNewSourceConfig] = useState('{}');
  const [creating, setCreating] = useState(false);

  const fetchSources = useCallback(async () => {
    try {
      const result = await window.mycelicMemory.sources?.list() ?? [];
      setSources(Array.isArray(result) ? result : []);
    } catch (err) {
      console.error('Failed to fetch sources:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSources();
  }, [fetchSources]);

  const selectSource = useCallback(async (source: DataSource) => {
    setSelectedSource(source);
    try {
      const [stats, history] = await Promise.all([
        window.mycelicMemory.sources?.stats(source.id).catch(() => null),
        window.mycelicMemory.sources?.history(source.id, 10).catch(() => []),
      ]);
      setSourceStats(stats);
      setSyncHistory(Array.isArray(history) ? history : []);
    } catch {
      setSourceStats(null);
      setSyncHistory([]);
    }
  }, []);

  const handleSync = async (sourceId: string) => {
    setSyncing(sourceId);
    try {
      await window.mycelicMemory.sources?.sync(sourceId);
      toast.success('Sync triggered');
      // Refresh after brief delay
      setTimeout(() => {
        fetchSources();
        if (selectedSource?.id === sourceId) selectSource(selectedSource);
      }, 2000);
    } catch (err) {
      toast.error('Sync failed: ' + (err instanceof Error ? err.message : 'Unknown error'));
    } finally {
      setSyncing(null);
    }
  };

  const handlePause = async (source: DataSource) => {
    try {
      if (source.status === 'paused') {
        await window.mycelicMemory.sources?.resume(source.id);
        toast.success(`${source.name} resumed`);
      } else {
        await window.mycelicMemory.sources?.pause(source.id);
        toast.success(`${source.name} paused`);
      }
      fetchSources();
    } catch (err) {
      toast.error('Failed to update source');
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await window.mycelicMemory.sources?.delete(deleteTarget.id);
      toast.success(`${deleteTarget.name} deleted`);
      if (selectedSource?.id === deleteTarget.id) {
        setSelectedSource(null);
        setSourceStats(null);
        setSyncHistory([]);
      }
      setDeleteTarget(null);
      fetchSources();
    } catch (err) {
      toast.error('Failed to delete source');
    }
  };

  const handleCreateSource = async () => {
    if (!newSourceName.trim()) {
      toast.error('Source name is required');
      return;
    }
    setCreating(true);
    try {
      await window.mycelicMemory.sources?.create({
        source_type: newSourceType,
        name: newSourceName,
        config: newSourceConfig,
      });
      toast.success('Source created');
      setShowAddDialog(false);
      setNewSourceName('');
      setNewSourceConfig('{}');
      fetchSources();
    } catch (err) {
      toast.error('Failed to create source: ' + (err instanceof Error ? err.message : 'Unknown error'));
    } finally {
      setCreating(false);
    }
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return 'Never';
    const d = new Date(dateStr);
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const getSourceTypeLabel = (type: string) =>
    SOURCE_TYPES.find((t) => t.value === type)?.label || type;

  const getSourceTypeIcon = (type: string) =>
    SOURCE_TYPES.find((t) => t.value === type)?.icon || '📦';

  if (loading) {
    return (
      <div className="p-8 flex items-center justify-center h-full">
        <div className="animate-spin w-8 h-8 border-4 border-primary-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  return (
    <div className="flex h-full animate-fade-in">
      {/* Source List */}
      <div className="w-96 border-r border-slate-700 flex flex-col">
        <div className="p-4 border-b border-slate-700">
          <div className="flex items-center justify-between mb-3">
            <h1 className="text-lg font-bold">Data Sources</h1>
            <div className="flex items-center gap-2">
              <button
                onClick={fetchSources}
                className="p-2 text-slate-400 hover:text-slate-200 hover:bg-slate-700 rounded-lg transition-colors"
                title="Refresh"
              >
                <RefreshCw className="w-4 h-4" />
              </button>
              <button
                onClick={() => setShowAddDialog(true)}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition-colors text-sm"
              >
                <Plus className="w-4 h-4" />
                Add Source
              </button>
            </div>
          </div>
          <p className="text-slate-500 text-xs">{sources.length} source{sources.length !== 1 ? 's' : ''} registered</p>
        </div>

        <div className="flex-1 overflow-auto p-2 space-y-1">
          {sources.length === 0 ? (
            <div className="text-center py-12">
              <Database className="w-12 h-12 text-slate-600 mx-auto mb-3" />
              <p className="text-slate-500 text-sm">No data sources configured</p>
              <p className="text-slate-600 text-xs mt-1">Add a source to start ingesting data</p>
            </div>
          ) : (
            sources.map((source) => (
              <button
                key={source.id}
                onClick={() => selectSource(source)}
                className={`w-full text-left p-3 rounded-lg transition-colors ${
                  selectedSource?.id === source.id
                    ? 'bg-primary-500/20 border border-primary-500/30'
                    : 'hover:bg-slate-700/50 border border-transparent'
                }`}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="text-lg shrink-0">{getSourceTypeIcon(source.source_type)}</span>
                    <div className="min-w-0">
                      <p className="text-sm font-medium truncate">{source.name}</p>
                      <p className="text-xs text-slate-500">{getSourceTypeLabel(source.source_type)}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <span className={`text-xs px-2 py-0.5 rounded-full border ${STATUS_COLORS[source.status]}`}>
                      {source.status}
                    </span>
                    <ChevronRight className="w-4 h-4 text-slate-600" />
                  </div>
                </div>
                {source.last_sync_at && (
                  <p className="text-xs text-slate-600 mt-1 ml-8">
                    Last sync: {formatDate(source.last_sync_at)}
                  </p>
                )}
              </button>
            ))
          )}
        </div>
      </div>

      {/* Source Detail */}
      <div className="flex-1 overflow-auto">
        {selectedSource ? (
          <div className="p-6">
            {/* Header */}
            <div className="flex items-center justify-between mb-6">
              <div className="flex items-center gap-3">
                <span className="text-2xl">{getSourceTypeIcon(selectedSource.source_type)}</span>
                <div>
                  <h2 className="text-xl font-bold">{selectedSource.name}</h2>
                  <p className="text-sm text-slate-400">{getSourceTypeLabel(selectedSource.source_type)}</p>
                </div>
                <span className={`text-xs px-2 py-0.5 rounded-full border ${STATUS_COLORS[selectedSource.status]}`}>
                  {selectedSource.status}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleSync(selectedSource.id)}
                  disabled={syncing === selectedSource.id || selectedSource.status === 'paused'}
                  className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-500/20 text-blue-400 rounded-lg hover:bg-blue-500/30 transition-colors text-sm disabled:opacity-50"
                >
                  {syncing === selectedSource.id ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <RefreshCw className="w-4 h-4" />
                  )}
                  Sync Now
                </button>
                <button
                  onClick={() => handlePause(selectedSource)}
                  className="flex items-center gap-1.5 px-3 py-1.5 bg-amber-500/20 text-amber-400 rounded-lg hover:bg-amber-500/30 transition-colors text-sm"
                >
                  {selectedSource.status === 'paused' ? (
                    <><Play className="w-4 h-4" /> Resume</>
                  ) : (
                    <><Pause className="w-4 h-4" /> Pause</>
                  )}
                </button>
                <button
                  onClick={() => setDeleteTarget(selectedSource)}
                  className="flex items-center gap-1.5 px-3 py-1.5 bg-red-500/20 text-red-400 rounded-lg hover:bg-red-500/30 transition-colors text-sm"
                >
                  <Trash2 className="w-4 h-4" />
                  Delete
                </button>
              </div>
            </div>

            {/* Stats Grid */}
            {sourceStats && (
              <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
                <div className="bg-slate-800 rounded-lg p-4 border border-slate-700">
                  <p className="text-slate-500 text-xs">Memories</p>
                  <p className="text-2xl font-bold mt-1">{sourceStats.total_memories}</p>
                </div>
                <div className="bg-slate-800 rounded-lg p-4 border border-slate-700">
                  <p className="text-slate-500 text-xs">Total Syncs</p>
                  <p className="text-2xl font-bold mt-1">{sourceStats.total_syncs}</p>
                </div>
                <div className="bg-slate-800 rounded-lg p-4 border border-slate-700">
                  <p className="text-slate-500 text-xs">Successful</p>
                  <p className="text-2xl font-bold mt-1 text-green-400">{sourceStats.successful_syncs}</p>
                </div>
                <div className="bg-slate-800 rounded-lg p-4 border border-slate-700">
                  <p className="text-slate-500 text-xs">Failed</p>
                  <p className="text-2xl font-bold mt-1 text-red-400">{sourceStats.failed_syncs}</p>
                </div>
              </div>
            )}

            {/* Config */}
            <div className="bg-slate-800 rounded-lg p-4 border border-slate-700 mb-6">
              <h3 className="text-sm font-semibold mb-2">Configuration</h3>
              <pre className="text-xs text-slate-400 bg-slate-900 rounded p-3 overflow-auto max-h-32">
                {(() => {
                  try {
                    return JSON.stringify(JSON.parse(selectedSource.config), null, 2);
                  } catch {
                    return selectedSource.config || '{}';
                  }
                })()}
              </pre>
              <div className="mt-2 text-xs text-slate-500">
                <span>Created: {formatDate(selectedSource.created_at)}</span>
                <span className="mx-2">|</span>
                <span>Updated: {formatDate(selectedSource.updated_at)}</span>
              </div>
            </div>

            {/* Sync History */}
            <div className="bg-slate-800 rounded-lg p-4 border border-slate-700">
              <h3 className="text-sm font-semibold mb-3">Sync History</h3>
              {syncHistory.length === 0 ? (
                <p className="text-slate-500 text-sm text-center py-4">No sync history yet</p>
              ) : (
                <div className="space-y-2">
                  {syncHistory.map((sync) => (
                    <div key={sync.id} className="flex items-center justify-between p-3 bg-slate-700/50 rounded-lg">
                      <div className="flex items-center gap-3">
                        {sync.status === 'running' ? (
                          <Loader2 className="w-4 h-4 text-blue-400 animate-spin shrink-0" />
                        ) : sync.status === 'completed' ? (
                          <CheckCircle className="w-4 h-4 text-green-400 shrink-0" />
                        ) : (
                          <AlertCircle className="w-4 h-4 text-red-400 shrink-0" />
                        )}
                        <div>
                          <p className={`text-sm font-medium ${SYNC_STATUS_COLORS[sync.status] || 'text-slate-300'}`}>
                            {sync.status.charAt(0).toUpperCase() + sync.status.slice(1)}
                          </p>
                          <div className="flex items-center gap-2 text-xs text-slate-500">
                            <Clock className="w-3 h-3" />
                            {formatDate(sync.started_at)}
                          </div>
                        </div>
                      </div>
                      <div className="text-right text-xs">
                        <p className="text-slate-400">{sync.items_processed} processed</p>
                        <p className="text-slate-500">
                          {sync.memories_created} created, {sync.duplicates_skipped} skipped
                        </p>
                        {sync.error && (
                          <p className="text-red-400 mt-1">{sync.error}</p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="flex items-center justify-center h-full">
            <div className="text-center">
              <Database className="w-16 h-16 text-slate-700 mx-auto mb-4" />
              <p className="text-slate-500">Select a data source to view details</p>
              <p className="text-slate-600 text-sm mt-1">Or add a new source to start ingesting data</p>
            </div>
          </div>
        )}
      </div>

      {/* Add Source Dialog */}
      {showAddDialog && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-slate-800 rounded-xl border border-slate-700 w-[480px] max-h-[80vh] overflow-auto">
            <div className="flex items-center justify-between p-4 border-b border-slate-700">
              <h2 className="font-semibold">Add Data Source</h2>
              <button
                onClick={() => setShowAddDialog(false)}
                className="p-1 hover:bg-slate-700 rounded"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
            <div className="p-4 space-y-4">
              <div>
                <label className="block text-sm text-slate-400 mb-1.5">Source Type</label>
                <select
                  value={newSourceType}
                  onChange={(e) => setNewSourceType(e.target.value)}
                  className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-sm focus:border-primary-500 focus:outline-none"
                >
                  {SOURCE_TYPES.map((t) => (
                    <option key={t.value} value={t.value}>
                      {t.icon} {t.label}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm text-slate-400 mb-1.5">Name</label>
                <input
                  type="text"
                  value={newSourceName}
                  onChange={(e) => setNewSourceName(e.target.value)}
                  placeholder="e.g., My Slack Workspace"
                  className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-sm focus:border-primary-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-sm text-slate-400 mb-1.5">Configuration (JSON)</label>
                <textarea
                  value={newSourceConfig}
                  onChange={(e) => setNewSourceConfig(e.target.value)}
                  rows={4}
                  className="w-full bg-slate-700 border border-slate-600 rounded-lg px-3 py-2 text-sm font-mono focus:border-primary-500 focus:outline-none"
                  placeholder='{"export_path": "/path/to/export"}'
                />
                <p className="text-xs text-slate-600 mt-1">
                  {newSourceType === 'slack' && 'For Slack: {"export_path": "/path/to/slack-export"}'}
                  {newSourceType === 'claude-code-local' && 'For Claude Code: {"claude_dir": "~/.claude"}'}
                  {newSourceType === 'discord' && 'For Discord: {"export_path": "/path/to/discord-export"}'}
                </p>
              </div>
            </div>
            <div className="flex justify-end gap-2 p-4 border-t border-slate-700">
              <button
                onClick={() => setShowAddDialog(false)}
                className="px-4 py-2 text-sm text-slate-400 hover:text-slate-200 rounded-lg hover:bg-slate-700 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateSource}
                disabled={creating || !newSourceName.trim()}
                className="flex items-center gap-1.5 px-4 py-2 text-sm bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition-colors disabled:opacity-50"
              >
                {creating && <Loader2 className="w-4 h-4 animate-spin" />}
                Create Source
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={!!deleteTarget}
        title="Delete Data Source"
        message={`Are you sure you want to delete "${deleteTarget?.name}"? This will not delete any memories that were already ingested.`}
        confirmLabel="Delete"
        variant="danger"
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  );
}

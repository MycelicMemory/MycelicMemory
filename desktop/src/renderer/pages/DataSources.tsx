import { useState, useEffect, useCallback } from 'react';
import {
  Database,
  Plus,
  RefreshCw,
  Pause,
  Play,
  Trash2,
  Settings,
  Activity,
  CheckCircle,
  XCircle,
  Clock,
  AlertCircle,
} from 'lucide-react';
import type {
  DataSource,
  DataSourceType,
  DataSourceStats,
  SyncHistoryEntry,
} from '../../shared/types';

const SOURCE_TYPE_ICONS: Record<DataSourceType, string> = {
  'claude-stream': '🤖',
  slack: '💬',
  email: '📧',
  browser: '🌐',
  notion: '📝',
  obsidian: '🔮',
  github: '🐙',
  custom: '⚡',
};

const SOURCE_TYPE_LABELS: Record<DataSourceType, string> = {
  'claude-stream': 'Claude Code',
  slack: 'Slack',
  email: 'Email',
  browser: 'Browser History',
  notion: 'Notion',
  obsidian: 'Obsidian',
  github: 'GitHub',
  custom: 'Custom',
};

const STATUS_COLORS = {
  active: 'bg-green-500/20 text-green-400',
  paused: 'bg-yellow-500/20 text-yellow-400',
  error: 'bg-red-500/20 text-red-400',
};

const SYNC_STATUS_COLORS = {
  running: 'text-blue-400',
  completed: 'text-green-400',
  failed: 'text-red-400',
};

interface SourceCardProps {
  source: DataSource;
  stats?: DataSourceStats;
  onPause: () => void;
  onResume: () => void;
  onSync: () => void;
  onDelete: () => void;
  onConfigure: () => void;
  isLoading: boolean;
}

function SourceCard({
  source,
  stats,
  onPause,
  onResume,
  onSync,
  onDelete,
  onConfigure,
  isLoading,
}: SourceCardProps) {
  return (
    <div className="bg-slate-800 rounded-xl border border-slate-700 p-6">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <span className="text-2xl">{SOURCE_TYPE_ICONS[source.source_type]}</span>
          <div>
            <h3 className="font-semibold text-white">{source.name}</h3>
            <p className="text-sm text-slate-400">
              {SOURCE_TYPE_LABELS[source.source_type]}
            </p>
          </div>
        </div>
        <span
          className={`px-2 py-1 rounded-full text-xs font-medium ${STATUS_COLORS[source.status]}`}
        >
          {source.status}
        </span>
      </div>

      {/* Stats */}
      {stats && (
        <div className="grid grid-cols-3 gap-4 mb-4">
          <div className="text-center p-2 bg-slate-700/50 rounded-lg">
            <div className="text-lg font-semibold text-white">
              {stats.total_memories}
            </div>
            <div className="text-xs text-slate-400">Memories</div>
          </div>
          <div className="text-center p-2 bg-slate-700/50 rounded-lg">
            <div className="text-lg font-semibold text-green-400">
              {stats.successful_syncs}
            </div>
            <div className="text-xs text-slate-400">Syncs</div>
          </div>
          <div className="text-center p-2 bg-slate-700/50 rounded-lg">
            <div className="text-lg font-semibold text-red-400">
              {stats.failed_syncs}
            </div>
            <div className="text-xs text-slate-400">Errors</div>
          </div>
        </div>
      )}

      {/* Last sync info */}
      <div className="mb-4 text-sm">
        {source.last_sync_at ? (
          <div className="flex items-center gap-2 text-slate-400">
            <Clock className="w-4 h-4" />
            Last sync: {new Date(source.last_sync_at).toLocaleString()}
          </div>
        ) : (
          <div className="flex items-center gap-2 text-slate-500">
            <Clock className="w-4 h-4" />
            Never synced
          </div>
        )}
      </div>

      {/* Error message */}
      {source.error_message && (
        <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-lg">
          <div className="flex items-start gap-2">
            <AlertCircle className="w-4 h-4 text-red-400 mt-0.5" />
            <p className="text-sm text-red-400">{source.error_message}</p>
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center gap-2 pt-4 border-t border-slate-700">
        {source.status === 'paused' ? (
          <button
            onClick={onResume}
            disabled={isLoading}
            className="flex items-center gap-1 px-3 py-1.5 bg-green-500/20 text-green-400 rounded-lg hover:bg-green-500/30 transition-colors text-sm"
          >
            <Play className="w-4 h-4" />
            Resume
          </button>
        ) : (
          <button
            onClick={onPause}
            disabled={isLoading}
            className="flex items-center gap-1 px-3 py-1.5 bg-yellow-500/20 text-yellow-400 rounded-lg hover:bg-yellow-500/30 transition-colors text-sm"
          >
            <Pause className="w-4 h-4" />
            Pause
          </button>
        )}
        <button
          onClick={onSync}
          disabled={isLoading || source.status === 'paused'}
          className="flex items-center gap-1 px-3 py-1.5 bg-blue-500/20 text-blue-400 rounded-lg hover:bg-blue-500/30 transition-colors text-sm disabled:opacity-50"
        >
          <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
          Sync
        </button>
        <button
          onClick={onConfigure}
          className="flex items-center gap-1 px-3 py-1.5 bg-slate-700 text-slate-300 rounded-lg hover:bg-slate-600 transition-colors text-sm"
        >
          <Settings className="w-4 h-4" />
          Configure
        </button>
        <button
          onClick={onDelete}
          disabled={isLoading}
          className="flex items-center gap-1 px-3 py-1.5 text-red-400 hover:bg-red-500/20 rounded-lg transition-colors text-sm ml-auto"
        >
          <Trash2 className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}

interface SyncHistoryListProps {
  history: SyncHistoryEntry[];
}

function SyncHistoryList({ history }: SyncHistoryListProps) {
  if (history.length === 0) {
    return (
      <div className="text-center py-8 text-slate-500">
        No sync history yet
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {history.map((entry) => (
        <div
          key={entry.id}
          className="flex items-center justify-between p-3 bg-slate-700/50 rounded-lg"
        >
          <div className="flex items-center gap-3">
            {entry.status === 'completed' && (
              <CheckCircle className="w-5 h-5 text-green-400" />
            )}
            {entry.status === 'failed' && (
              <XCircle className="w-5 h-5 text-red-400" />
            )}
            {entry.status === 'running' && (
              <RefreshCw className="w-5 h-5 text-blue-400 animate-spin" />
            )}
            <div>
              <p className={`text-sm ${SYNC_STATUS_COLORS[entry.status]}`}>
                {entry.status === 'completed'
                  ? `Created ${entry.memories_created} memories (${entry.duplicates_skipped} duplicates)`
                  : entry.status === 'failed'
                  ? entry.error || 'Sync failed'
                  : 'Sync in progress...'}
              </p>
              <p className="text-xs text-slate-500">
                {new Date(entry.started_at).toLocaleString()}
              </p>
            </div>
          </div>
          <div className="text-sm text-slate-400">
            {entry.items_processed} items
          </div>
        </div>
      ))}
    </div>
  );
}

interface AddSourceModalProps {
  isOpen: boolean;
  onClose: () => void;
  onAdd: (type: DataSourceType, name: string) => void;
}

function AddSourceModal({ isOpen, onClose, onAdd }: AddSourceModalProps) {
  const [selectedType, setSelectedType] = useState<DataSourceType>('claude-stream');
  const [name, setName] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (name.trim()) {
      onAdd(selectedType, name.trim());
      setName('');
      onClose();
    }
  };

  if (!isOpen) return null;

  const sourceTypes: DataSourceType[] = [
    'claude-stream',
    'slack',
    'email',
    'browser',
    'notion',
    'obsidian',
    'github',
    'custom',
  ];

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-slate-800 rounded-xl border border-slate-700 p-6 w-full max-w-md">
        <h2 className="text-xl font-semibold text-white mb-4">Add Data Source</h2>
        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium text-slate-300 mb-2">
              Source Type
            </label>
            <div className="grid grid-cols-2 gap-2">
              {sourceTypes.map((type) => (
                <button
                  key={type}
                  type="button"
                  onClick={() => setSelectedType(type)}
                  className={`flex items-center gap-2 p-3 rounded-lg border transition-colors ${
                    selectedType === type
                      ? 'border-primary-500 bg-primary-500/20'
                      : 'border-slate-600 hover:border-slate-500'
                  }`}
                >
                  <span>{SOURCE_TYPE_ICONS[type]}</span>
                  <span className="text-sm text-white">{SOURCE_TYPE_LABELS[type]}</span>
                </button>
              ))}
            </div>
          </div>
          <div className="mb-6">
            <label className="block text-sm font-medium text-slate-300 mb-2">
              Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={`My ${SOURCE_TYPE_LABELS[selectedType]}`}
              className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
            />
          </div>
          <div className="flex gap-3">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 bg-slate-700 text-slate-300 rounded-lg hover:bg-slate-600 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!name.trim()}
              className="flex-1 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition-colors disabled:opacity-50"
            >
              Add Source
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default function DataSources() {
  const [sources, setSources] = useState<DataSource[]>([]);
  const [sourceStats, setSourceStats] = useState<Record<string, DataSourceStats>>({});
  const [selectedSource, setSelectedSource] = useState<string | null>(null);
  const [syncHistory, setSyncHistory] = useState<SyncHistoryEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);

  const loadSources = useCallback(async () => {
    try {
      setIsLoading(true);
      const data = await window.mycelicMemory.sources.list();
      setSources(data);

      // Load stats for each source
      const stats: Record<string, DataSourceStats> = {};
      for (const source of data) {
        try {
          stats[source.id] = await window.mycelicMemory.sources.stats(source.id);
        } catch {
          // Ignore stats errors
        }
      }
      setSourceStats(stats);
    } catch (error) {
      console.error('Failed to load sources:', error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const loadSyncHistory = useCallback(async (sourceId: string) => {
    try {
      const history = await window.mycelicMemory.sources.history(sourceId, 10);
      setSyncHistory(history);
    } catch (error) {
      console.error('Failed to load sync history:', error);
    }
  }, []);

  useEffect(() => {
    loadSources();
  }, [loadSources]);

  useEffect(() => {
    if (selectedSource) {
      loadSyncHistory(selectedSource);
    }
  }, [selectedSource, loadSyncHistory]);

  const handleAddSource = async (type: DataSourceType, name: string) => {
    try {
      await window.mycelicMemory.sources.create({
        source_type: type,
        name,
      });
      await loadSources();
    } catch (error) {
      console.error('Failed to add source:', error);
    }
  };

  const handlePause = async (id: string) => {
    try {
      setActionLoading(id);
      await window.mycelicMemory.sources.pause(id);
      await loadSources();
    } catch (error) {
      console.error('Failed to pause source:', error);
    } finally {
      setActionLoading(null);
    }
  };

  const handleResume = async (id: string) => {
    try {
      setActionLoading(id);
      await window.mycelicMemory.sources.resume(id);
      await loadSources();
    } catch (error) {
      console.error('Failed to resume source:', error);
    } finally {
      setActionLoading(null);
    }
  };

  const handleSync = async (id: string) => {
    try {
      setActionLoading(id);
      await window.mycelicMemory.sources.sync(id);
      await loadSources();
      if (selectedSource === id) {
        await loadSyncHistory(id);
      }
    } catch (error) {
      console.error('Failed to trigger sync:', error);
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this data source? This cannot be undone.')) {
      return;
    }
    try {
      setActionLoading(id);
      await window.mycelicMemory.sources.delete(id);
      if (selectedSource === id) {
        setSelectedSource(null);
      }
      await loadSources();
    } catch (error) {
      console.error('Failed to delete source:', error);
    } finally {
      setActionLoading(null);
    }
  };

  return (
    <div className="p-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-white mb-2">Data Sources</h1>
          <p className="text-slate-400">
            Manage your memory ingestion sources - Slack, Claude Code, Email, and more
          </p>
        </div>
        <button
          onClick={() => setShowAddModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition-colors"
        >
          <Plus className="w-5 h-5" />
          Add Source
        </button>
      </div>

      <div className="flex gap-8">
        {/* Sources List */}
        <div className="flex-1">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <RefreshCw className="w-8 h-8 text-primary-400 animate-spin" />
            </div>
          ) : sources.length === 0 ? (
            <div className="text-center py-12">
              <Database className="w-16 h-16 text-slate-600 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-white mb-2">
                No data sources configured
              </h3>
              <p className="text-slate-400 mb-6">
                Add your first data source to start ingesting memories
              </p>
              <button
                onClick={() => setShowAddModal(true)}
                className="inline-flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition-colors"
              >
                <Plus className="w-5 h-5" />
                Add Data Source
              </button>
            </div>
          ) : (
            <div className="grid gap-4">
              {sources.map((source) => (
                <div
                  key={source.id}
                  onClick={() => setSelectedSource(source.id)}
                  className={`cursor-pointer transition-all ${
                    selectedSource === source.id ? 'ring-2 ring-primary-500' : ''
                  }`}
                >
                  <SourceCard
                    source={source}
                    stats={sourceStats[source.id]}
                    onPause={() => handlePause(source.id)}
                    onResume={() => handleResume(source.id)}
                    onSync={() => handleSync(source.id)}
                    onDelete={() => handleDelete(source.id)}
                    onConfigure={() => {
                      // TODO: Open configuration modal
                      alert('Configuration coming soon!');
                    }}
                    isLoading={actionLoading === source.id}
                  />
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Sync History Sidebar */}
        {selectedSource && (
          <div className="w-96">
            <div className="bg-slate-800 rounded-xl border border-slate-700 p-6">
              <div className="flex items-center gap-2 mb-4">
                <Activity className="w-5 h-5 text-primary-400" />
                <h2 className="text-lg font-semibold text-white">Sync History</h2>
              </div>
              <SyncHistoryList history={syncHistory} />
            </div>
          </div>
        )}
      </div>

      {/* Add Source Modal */}
      <AddSourceModal
        isOpen={showAddModal}
        onClose={() => setShowAddModal(false)}
        onAdd={handleAddSource}
      />
    </div>
  );
}

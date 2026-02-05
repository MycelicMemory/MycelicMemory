import { useState, useEffect } from 'react';
import {
  Search,
  Filter,
  ChevronRight,
  X,
  Edit2,
  Trash2,
  Tag,
  Calendar,
  Star,
} from 'lucide-react';
import type { Memory, Domain, SearchResult } from '../../shared/types';

interface MemoryCardProps {
  memory: Memory;
  onClick: (memory: Memory) => void;
  isSelected: boolean;
}

function MemoryCard({ memory, onClick, isSelected }: MemoryCardProps) {
  return (
    <div
      onClick={() => onClick(memory)}
      className={`p-4 rounded-lg cursor-pointer transition-all ${
        isSelected
          ? 'bg-primary-500/20 border-2 border-primary-500'
          : 'bg-slate-800 border border-slate-700 hover:border-slate-600'
      }`}
    >
      <p className="text-sm line-clamp-3 mb-3">{memory.content}</p>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-xs bg-slate-700 px-2 py-1 rounded">{memory.domain || 'general'}</span>
          <span className="text-xs text-slate-500 flex items-center gap-1">
            <Star className="w-3 h-3" />
            {memory.importance}
          </span>
        </div>
        <ChevronRight className="w-4 h-4 text-slate-500" />
      </div>
    </div>
  );
}

interface MemoryDetailProps {
  memory: Memory;
  onClose: () => void;
  onUpdate: (id: string, data: { content?: string; importance?: number }) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

function MemoryDetail({ memory, onClose, onUpdate, onDelete }: MemoryDetailProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editedContent, setEditedContent] = useState(memory.content);
  const [editedImportance, setEditedImportance] = useState(memory.importance);

  useEffect(() => {
    setEditedContent(memory.content);
    setEditedImportance(memory.importance);
    setIsEditing(false);
  }, [memory]);

  const handleSave = async () => {
    await onUpdate(memory.id, {
      content: editedContent,
      importance: editedImportance,
    });
    setIsEditing(false);
  };

  const handleDelete = async () => {
    if (confirm('Are you sure you want to delete this memory?')) {
      await onDelete(memory.id);
    }
  };

  return (
    <div className="bg-slate-800 rounded-xl border border-slate-700 h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-slate-700">
        <h3 className="font-semibold">Memory Details</h3>
        <div className="flex items-center gap-2">
          {!isEditing && (
            <button
              onClick={() => setIsEditing(true)}
              className="p-2 hover:bg-slate-700 rounded-lg transition-colors"
            >
              <Edit2 className="w-4 h-4" />
            </button>
          )}
          <button
            onClick={handleDelete}
            className="p-2 hover:bg-red-500/20 text-red-400 rounded-lg transition-colors"
          >
            <Trash2 className="w-4 h-4" />
          </button>
          <button
            onClick={onClose}
            className="p-2 hover:bg-slate-700 rounded-lg transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-4 space-y-4">
        {/* Content Field */}
        <div>
          <label className="text-xs text-slate-400 uppercase tracking-wide">Content</label>
          {isEditing ? (
            <textarea
              value={editedContent}
              onChange={(e) => setEditedContent(e.target.value)}
              className="w-full mt-2 p-3 bg-slate-700 rounded-lg text-sm resize-none h-40 focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          ) : (
            <p className="mt-2 text-sm leading-relaxed whitespace-pre-wrap">{memory.content}</p>
          )}
        </div>

        {/* Metadata */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-xs text-slate-400 uppercase tracking-wide">Domain</label>
            <p className="mt-1 text-sm bg-slate-700 px-3 py-2 rounded-lg">
              {memory.domain || 'general'}
            </p>
          </div>
          <div>
            <label className="text-xs text-slate-400 uppercase tracking-wide">Importance</label>
            {isEditing ? (
              <input
                type="number"
                min="1"
                max="10"
                value={editedImportance}
                onChange={(e) => setEditedImportance(parseInt(e.target.value) || 5)}
                className="w-full mt-1 p-2 bg-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            ) : (
              <p className="mt-1 text-sm bg-slate-700 px-3 py-2 rounded-lg">{memory.importance}/10</p>
            )}
          </div>
        </div>

        {/* Tags */}
        {memory.tags && memory.tags.length > 0 && (
          <div>
            <label className="text-xs text-slate-400 uppercase tracking-wide flex items-center gap-1">
              <Tag className="w-3 h-3" /> Tags
            </label>
            <div className="flex flex-wrap gap-2 mt-2">
              {memory.tags.map((tag) => (
                <span key={tag} className="text-xs bg-primary-500/20 text-primary-400 px-2 py-1 rounded">
                  {tag}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Timestamps */}
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <label className="text-xs text-slate-400 uppercase tracking-wide flex items-center gap-1">
              <Calendar className="w-3 h-3" /> Created
            </label>
            <p className="mt-1 text-slate-300">{new Date(memory.created_at).toLocaleDateString()}</p>
          </div>
          <div>
            <label className="text-xs text-slate-400 uppercase tracking-wide">Updated</label>
            <p className="mt-1 text-slate-300">{new Date(memory.updated_at).toLocaleDateString()}</p>
          </div>
        </div>

        {/* ID */}
        <div>
          <label className="text-xs text-slate-400 uppercase tracking-wide">ID</label>
          <p className="mt-1 text-xs font-mono text-slate-500 break-all">{memory.id}</p>
        </div>
      </div>

      {/* Actions */}
      {isEditing && (
        <div className="p-4 border-t border-slate-700 flex gap-2">
          <button
            onClick={() => setIsEditing(false)}
            className="flex-1 py-2 px-4 bg-slate-700 rounded-lg hover:bg-slate-600 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            className="flex-1 py-2 px-4 bg-primary-500 rounded-lg hover:bg-primary-600 transition-colors"
          >
            Save
          </button>
        </div>
      )}
    </div>
  );
}

export default function MemoryBrowser() {
  const [memories, setMemories] = useState<Memory[]>([]);
  const [selectedMemory, setSelectedMemory] = useState<Memory | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchType, setSearchType] = useState<'keyword' | 'semantic'>('keyword');
  const [loading, setLoading] = useState(true);
  const [filters, setFilters] = useState({
    domain: '',
    minImportance: 1,
    maxImportance: 10,
  });
  const [showFilters, setShowFilters] = useState(false);
  const [domains, setDomains] = useState<Domain[]>([]);

  useEffect(() => {
    fetchMemories();
    fetchDomains();
  }, []);

  async function fetchMemories() {
    try {
      setLoading(true);
      const response = await window.mycelicMemory.memory.list({ limit: 50 });
      setMemories(response || []);
    } catch (err) {
      console.error('Failed to fetch memories:', err);
    } finally {
      setLoading(false);
    }
  }

  async function fetchDomains() {
    try {
      const response = await window.mycelicMemory.domains.list();
      setDomains(response || []);
    } catch (err) {
      console.error('Failed to fetch domains:', err);
    }
  }

  async function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    if (!searchQuery.trim()) {
      fetchMemories();
      return;
    }

    try {
      setLoading(true);
      const response = await window.mycelicMemory.memory.search({
        query: searchQuery,
        search_type: searchType,
        domain: filters.domain || undefined,
      });
      setMemories(response.map((r: SearchResult) => r.memory));
    } catch (err) {
      console.error('Search failed:', err);
    } finally {
      setLoading(false);
    }
  }

  async function handleUpdate(id: string, data: { content?: string; importance?: number }) {
    try {
      await window.mycelicMemory.memory.update(id, data);
      fetchMemories();
      if (selectedMemory?.id === id) {
        setSelectedMemory({ ...selectedMemory, ...data });
      }
    } catch (err) {
      console.error('Update failed:', err);
    }
  }

  async function handleDelete(id: string) {
    try {
      await window.mycelicMemory.memory.delete(id);
      setSelectedMemory(null);
      fetchMemories();
    } catch (err) {
      console.error('Delete failed:', err);
    }
  }

  const filteredMemories = memories.filter((m) => {
    if (filters.domain && m.domain !== filters.domain) return false;
    if (m.importance < filters.minImportance || m.importance > filters.maxImportance) return false;
    return true;
  });

  return (
    <div className="h-screen flex">
      {/* Memory List */}
      <div className="w-96 border-r border-slate-700 flex flex-col bg-slate-900">
        {/* Search */}
        <div className="p-4 border-b border-slate-700">
          <form onSubmit={handleSearch}>
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
              <input
                type="text"
                placeholder="Search memories..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-10 pr-4 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </div>
          </form>

          <div className="flex items-center justify-between mt-3">
            <div className="flex gap-2">
              <button
                onClick={() => setSearchType('keyword')}
                className={`text-xs px-3 py-1 rounded-full transition-colors ${
                  searchType === 'keyword'
                    ? 'bg-primary-500 text-white'
                    : 'bg-slate-700 text-slate-400 hover:bg-slate-600'
                }`}
              >
                Keyword
              </button>
              <button
                onClick={() => setSearchType('semantic')}
                className={`text-xs px-3 py-1 rounded-full transition-colors ${
                  searchType === 'semantic'
                    ? 'bg-primary-500 text-white'
                    : 'bg-slate-700 text-slate-400 hover:bg-slate-600'
                }`}
              >
                Semantic
              </button>
            </div>
            <button
              onClick={() => setShowFilters(!showFilters)}
              className={`p-2 rounded-lg transition-colors ${
                showFilters ? 'bg-primary-500/20 text-primary-400' : 'hover:bg-slate-700'
              }`}
            >
              <Filter className="w-4 h-4" />
            </button>
          </div>

          {/* Filters Panel */}
          {showFilters && (
            <div className="mt-3 p-3 bg-slate-800 rounded-lg space-y-3">
              <div>
                <label className="text-xs text-slate-400">Domain</label>
                <select
                  value={filters.domain}
                  onChange={(e) => setFilters({ ...filters, domain: e.target.value })}
                  className="w-full mt-1 p-2 bg-slate-700 rounded-lg text-sm focus:outline-none"
                >
                  <option value="">All domains</option>
                  {domains.map((d) => (
                    <option key={d.id || d.name} value={d.name}>
                      {d.name}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-xs text-slate-400">
                  Importance: {filters.minImportance} - {filters.maxImportance}
                </label>
                <div className="flex gap-2 mt-1">
                  <input
                    type="range"
                    min="1"
                    max="10"
                    value={filters.minImportance}
                    onChange={(e) => setFilters({ ...filters, minImportance: parseInt(e.target.value) })}
                    className="flex-1"
                  />
                  <input
                    type="range"
                    min="1"
                    max="10"
                    value={filters.maxImportance}
                    onChange={(e) => setFilters({ ...filters, maxImportance: parseInt(e.target.value) })}
                    className="flex-1"
                  />
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Memory List */}
        <div className="flex-1 overflow-auto p-4 space-y-3">
          {loading ? (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin w-6 h-6 border-2 border-primary-500 border-t-transparent rounded-full" />
            </div>
          ) : filteredMemories.length > 0 ? (
            filteredMemories.map((memory) => (
              <MemoryCard
                key={memory.id}
                memory={memory}
                onClick={setSelectedMemory}
                isSelected={selectedMemory?.id === memory.id}
              />
            ))
          ) : (
            <p className="text-center text-slate-500 py-8">No memories found</p>
          )}
        </div>

        {/* Count */}
        <div className="p-4 border-t border-slate-700 text-sm text-slate-400">
          {filteredMemories.length} memories
        </div>
      </div>

      {/* Detail Panel */}
      <div className="flex-1 p-4 bg-slate-900">
        {selectedMemory ? (
          <MemoryDetail
            memory={selectedMemory}
            onClose={() => setSelectedMemory(null)}
            onUpdate={handleUpdate}
            onDelete={handleDelete}
          />
        ) : (
          <div className="h-full flex items-center justify-center text-slate-500">
            <div className="text-center">
              <Search className="w-12 h-12 mx-auto mb-4 opacity-50" />
              <p>Select a memory to view details</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

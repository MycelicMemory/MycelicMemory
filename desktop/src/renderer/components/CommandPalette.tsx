import { useState, useEffect, useRef, useCallback } from 'react';
import { Search, Brain, MessageSquare, Globe, X } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import type { Memory, SearchResult } from '../../shared/types';

interface CommandPaletteProps {
  open: boolean;
  onClose: () => void;
}

interface SearchResultItem {
  type: 'memory' | 'session' | 'domain';
  id: string;
  title: string;
  subtitle: string;
  route: string;
}

export function CommandPalette({ open, onClose }: CommandPaletteProps) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResultItem[]>([]);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [loading, setLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    if (open) {
      setQuery('');
      setResults([]);
      setSelectedIndex(0);
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [open]);

  const doSearch = useCallback(async (q: string) => {
    if (!q.trim()) {
      setResults([]);
      return;
    }

    setLoading(true);
    try {
      const [memoriesRes, sessionsRes, domainsRes] = await Promise.allSettled([
        window.mycelicMemory.memory.search({ query: q, search_type: 'keyword', limit: 5 }),
        window.mycelicMemory.claude.search(q).catch(() => []),
        window.mycelicMemory.domains.list().catch(() => []),
      ]);

      const items: SearchResultItem[] = [];

      if (memoriesRes.status === 'fulfilled' && memoriesRes.value) {
        for (const r of memoriesRes.value.slice(0, 5)) {
          const mem = (r as SearchResult).memory || (r as unknown as Memory);
          if (mem?.id) {
            items.push({
              type: 'memory',
              id: mem.id,
              title: mem.content?.slice(0, 80) || 'Memory',
              subtitle: `${mem.domain || 'general'} | importance ${mem.importance}`,
              route: '/memories',
            });
          }
        }
      }

      if (sessionsRes.status === 'fulfilled' && Array.isArray(sessionsRes.value)) {
        for (const s of sessionsRes.value.slice(0, 3)) {
          items.push({
            type: 'session',
            id: s.id || s.session_id,
            title: s.title || s.first_prompt?.slice(0, 60) || 'Session',
            subtitle: `${s.message_count || 0} messages`,
            route: '/sessions',
          });
        }
      }

      if (domainsRes.status === 'fulfilled' && Array.isArray(domainsRes.value)) {
        const filtered = domainsRes.value.filter((d: { name: string }) =>
          d.name.toLowerCase().includes(q.toLowerCase())
        );
        for (const d of filtered.slice(0, 2)) {
          items.push({
            type: 'domain',
            id: d.name,
            title: d.name,
            subtitle: 'Domain',
            route: '/memories',
          });
        }
      }

      setResults(items);
      setSelectedIndex(0);
    } catch (err) {
      console.error('Command palette search failed:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timeout = setTimeout(() => doSearch(query), 200);
    return () => clearTimeout(timeout);
  }, [query, doSearch]);

  const selectResult = (item: SearchResultItem) => {
    navigate(item.route);
    onClose();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onClose();
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSelectedIndex((i) => Math.min(i + 1, results.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSelectedIndex((i) => Math.max(i - 1, 0));
    } else if (e.key === 'Enter' && results[selectedIndex]) {
      selectResult(results[selectedIndex]);
    }
  };

  if (!open) return null;

  const iconMap = {
    memory: Brain,
    session: MessageSquare,
    domain: Globe,
  };

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-24">
      <div className="absolute inset-0 bg-black/60" onClick={onClose} />
      <div
        className="relative bg-slate-800 border border-slate-700 rounded-xl shadow-2xl max-w-lg w-full mx-4 animate-fade-in overflow-hidden"
        onKeyDown={handleKeyDown}
      >
        {/* Search Input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-slate-700">
          <Search className="w-5 h-5 text-slate-400 shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search memories, sessions, domains..."
            className="flex-1 bg-transparent text-sm focus:outline-none"
          />
          {query && (
            <button onClick={() => setQuery('')} className="p-1 hover:bg-slate-700 rounded">
              <X className="w-3 h-3 text-slate-400" />
            </button>
          )}
          <kbd className="text-xs text-slate-500 bg-slate-700 px-1.5 py-0.5 rounded">ESC</kbd>
        </div>

        {/* Results */}
        {results.length > 0 && (
          <div className="max-h-80 overflow-auto py-2">
            {results.map((item, i) => {
              const Icon = iconMap[item.type];
              return (
                <button
                  key={`${item.type}-${item.id}`}
                  onClick={() => selectResult(item)}
                  className={`w-full flex items-center gap-3 px-4 py-2.5 text-left transition-colors ${
                    i === selectedIndex ? 'bg-primary-500/20 text-primary-300' : 'hover:bg-slate-700'
                  }`}
                >
                  <Icon className="w-4 h-4 text-slate-400 shrink-0" />
                  <div className="min-w-0 flex-1">
                    <p className="text-sm truncate">{item.title}</p>
                    <p className="text-xs text-slate-500 truncate">{item.subtitle}</p>
                  </div>
                  <span className="text-xs text-slate-600 capitalize shrink-0">{item.type}</span>
                </button>
              );
            })}
          </div>
        )}

        {/* Loading / Empty */}
        {loading && (
          <div className="px-4 py-6 text-center text-sm text-slate-500">Searching...</div>
        )}
        {!loading && query && results.length === 0 && (
          <div className="px-4 py-6 text-center text-sm text-slate-500">No results found</div>
        )}

        {/* Hints */}
        {!query && (
          <div className="px-4 py-4 text-xs text-slate-500 space-y-1">
            <p>Type to search across memories, sessions, and domains</p>
            <p><kbd className="bg-slate-700 px-1 rounded">Up/Down</kbd> to navigate, <kbd className="bg-slate-700 px-1 rounded">Enter</kbd> to select</p>
          </div>
        )}
      </div>
    </div>
  );
}

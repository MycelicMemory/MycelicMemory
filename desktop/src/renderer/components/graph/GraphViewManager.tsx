import { useState, useRef, useEffect } from 'react';
import { Bookmark, Pencil, Trash2, Check, X } from 'lucide-react';
import type { GraphView, GraphFilterState, GraphPhysicsSettings, GraphStyleSettings } from '../../../shared/types';

interface GraphViewManagerProps {
  views: GraphView[];
  activeViewId?: string;
  currentFilter: GraphFilterState;
  currentPhysics: GraphPhysicsSettings;
  currentStyle: GraphStyleSettings;
  hiddenNodeIds: string[];
  pinnedMemoryIds: string[];
  onLoadView: (view: GraphView) => void;
  onSave: (view: GraphView) => Promise<GraphView>;
  onDelete: (id: string) => Promise<void>;
}

export default function GraphViewManager({
  views, activeViewId, currentFilter, currentPhysics, currentStyle,
  hiddenNodeIds, pinnedMemoryIds, onLoadView, onSave, onDelete,
}: GraphViewManagerProps) {
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [newName, setNewName] = useState('');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState('');
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setOpen(false);
        setSaving(false);
        setEditingId(null);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  async function handleSave() {
    if (!newName.trim()) return;
    const now = new Date().toISOString();
    const view: GraphView = {
      id: crypto.randomUUID(),
      name: newName.trim(),
      created_at: now,
      updated_at: now,
      filter: currentFilter,
      physics: currentPhysics,
      style: currentStyle,
      hiddenNodeIds,
      pinnedMemoryIds,
    };
    await onSave(view);
    setNewName('');
    setSaving(false);
  }

  async function handleRename(id: string) {
    if (!editName.trim()) return;
    const view = views.find((v) => v.id === id);
    if (!view) return;
    await onSave({ ...view, name: editName.trim() });
    setEditingId(null);
  }

  return (
    <div className="relative" ref={menuRef}>
      <button
        onClick={() => setOpen(!open)}
        className={`px-3 py-1.5 rounded-lg text-sm flex items-center gap-2 transition-colors ${
          activeViewId
            ? 'bg-indigo-500/20 text-indigo-400 border border-indigo-500/50'
            : 'bg-slate-800 text-slate-400 border border-slate-700'
        }`}
        title="Saved Views"
      >
        <Bookmark className="w-4 h-4" />
        Views
      </button>

      {open && (
        <div className="absolute right-0 top-full mt-1 w-64 bg-slate-700 border border-slate-600 rounded-lg shadow-xl z-50 overflow-hidden">
          <div className="px-3 py-2 border-b border-slate-600 text-xs uppercase text-slate-400">
            Saved Views
          </div>

          {views.length === 0 ? (
            <div className="px-3 py-3 text-sm text-slate-500 text-center">No saved views</div>
          ) : (
            <div className="max-h-48 overflow-y-auto">
              {views.map((view) => (
                <div
                  key={view.id}
                  className={`group flex items-center px-3 py-2 hover:bg-slate-600/50 ${
                    view.id === activeViewId ? 'bg-indigo-500/10' : ''
                  }`}
                >
                  {editingId === view.id ? (
                    <div className="flex-1 flex items-center gap-1">
                      <input
                        autoFocus
                        value={editName}
                        onChange={(e) => setEditName(e.target.value)}
                        onKeyDown={(e) => { if (e.key === 'Enter') handleRename(view.id); if (e.key === 'Escape') setEditingId(null); }}
                        className="flex-1 px-1.5 py-0.5 bg-slate-800 border border-slate-500 rounded text-sm"
                      />
                      <button onClick={() => handleRename(view.id)} className="p-0.5 hover:text-green-400">
                        <Check className="w-3.5 h-3.5" />
                      </button>
                      <button onClick={() => setEditingId(null)} className="p-0.5 hover:text-red-400">
                        <X className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  ) : (
                    <>
                      <button
                        onClick={() => { onLoadView(view); setOpen(false); }}
                        className="flex-1 text-left text-sm text-slate-200 truncate"
                      >
                        {view.id === activeViewId && <span className="text-indigo-400 mr-1">*</span>}
                        {view.name}
                      </button>
                      <div className="hidden group-hover:flex items-center gap-0.5 ml-1">
                        <button
                          onClick={() => { setEditingId(view.id); setEditName(view.name); }}
                          className="p-1 hover:text-blue-400"
                        >
                          <Pencil className="w-3 h-3" />
                        </button>
                        <button
                          onClick={() => onDelete(view.id)}
                          className="p-1 hover:text-red-400"
                        >
                          <Trash2 className="w-3 h-3" />
                        </button>
                      </div>
                    </>
                  )}
                </div>
              ))}
            </div>
          )}

          <div className="border-t border-slate-600">
            {saving ? (
              <div className="px-3 py-2 flex items-center gap-1">
                <input
                  autoFocus
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  onKeyDown={(e) => { if (e.key === 'Enter') handleSave(); if (e.key === 'Escape') setSaving(false); }}
                  placeholder="View name..."
                  className="flex-1 px-2 py-1 bg-slate-800 border border-slate-500 rounded text-sm"
                />
                <button onClick={handleSave} className="p-1 hover:text-green-400">
                  <Check className="w-4 h-4" />
                </button>
                <button onClick={() => setSaving(false)} className="p-1 hover:text-red-400">
                  <X className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <button
                onClick={() => setSaving(true)}
                className="w-full px-3 py-2 text-sm text-slate-300 hover:bg-slate-600/50 text-left"
              >
                Save Current View...
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

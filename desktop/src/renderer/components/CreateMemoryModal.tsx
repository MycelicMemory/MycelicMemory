import { useState, useEffect, useRef } from 'react';
import { X, Plus, Tag } from 'lucide-react';
import { toast } from './Toast';

interface CreateMemoryModalProps {
  open: boolean;
  onClose: () => void;
  onCreated?: () => void;
}

export function CreateMemoryModal({ open, onClose, onCreated }: CreateMemoryModalProps) {
  const [content, setContent] = useState('');
  const [domain, setDomain] = useState('');
  const [importance, setImportance] = useState(5);
  const [tagInput, setTagInput] = useState('');
  const [tags, setTags] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const contentRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (open) {
      setContent('');
      setDomain('');
      setImportance(5);
      setTags([]);
      setTagInput('');
      setTimeout(() => contentRef.current?.focus(), 100);
    }
  }, [open]);

  if (!open) return null;

  const addTag = () => {
    const trimmed = tagInput.trim().toLowerCase();
    if (trimmed && !tags.includes(trimmed)) {
      setTags([...tags, trimmed]);
    }
    setTagInput('');
  };

  const removeTag = (tag: string) => {
    setTags(tags.filter((t) => t !== tag));
  };

  const handleSubmit = async () => {
    if (!content.trim()) {
      toast.error('Content is required');
      return;
    }

    setSaving(true);
    try {
      await window.mycelicMemory.memory.store({
        content: content.trim(),
        domain: domain || undefined,
        importance,
        tags: tags.length > 0 ? tags : undefined,
      });
      toast.success('Memory stored');
      onCreated?.();
      onClose();
    } catch (err) {
      toast.error('Failed to store memory');
      console.error('Store failed:', err);
    } finally {
      setSaving(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') onClose();
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) handleSubmit();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" onKeyDown={handleKeyDown}>
      <div className="absolute inset-0 bg-black/60" onClick={onClose} />
      <div className="relative bg-slate-800 border border-slate-700 rounded-xl shadow-2xl max-w-lg w-full mx-4 animate-fade-in">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-700">
          <h3 className="font-semibold text-slate-200">Create Memory</h3>
          <button onClick={onClose} className="p-1 hover:bg-slate-700 rounded-lg transition-colors">
            <X className="w-4 h-4 text-slate-400" />
          </button>
        </div>

        {/* Body */}
        <div className="p-4 space-y-4">
          <div>
            <label className="text-xs text-slate-400 uppercase tracking-wide">Content</label>
            <textarea
              ref={contentRef}
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder="What do you want to remember?"
              className="w-full mt-1 p-3 bg-slate-700 rounded-lg text-sm resize-none h-32 focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs text-slate-400 uppercase tracking-wide">Domain</label>
              <input
                type="text"
                value={domain}
                onChange={(e) => setDomain(e.target.value)}
                placeholder="general"
                className="w-full mt-1 p-2 bg-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </div>
            <div>
              <label className="text-xs text-slate-400 uppercase tracking-wide">
                Importance ({importance})
              </label>
              <input
                type="range"
                min="1"
                max="10"
                value={importance}
                onChange={(e) => setImportance(parseInt(e.target.value))}
                className="w-full mt-2"
              />
            </div>
          </div>

          <div>
            <label className="text-xs text-slate-400 uppercase tracking-wide flex items-center gap-1">
              <Tag className="w-3 h-3" /> Tags
            </label>
            <div className="flex gap-2 mt-1">
              <input
                type="text"
                value={tagInput}
                onChange={(e) => setTagInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    addTag();
                  }
                }}
                placeholder="Add tag..."
                className="flex-1 p-2 bg-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
              <button
                onClick={addTag}
                className="p-2 bg-slate-700 hover:bg-slate-600 rounded-lg transition-colors"
              >
                <Plus className="w-4 h-4" />
              </button>
            </div>
            {tags.length > 0 && (
              <div className="flex flex-wrap gap-2 mt-2">
                {tags.map((tag) => (
                  <span
                    key={tag}
                    className="text-xs bg-primary-500/20 text-primary-400 px-2 py-1 rounded flex items-center gap-1"
                  >
                    {tag}
                    <button onClick={() => removeTag(tag)} className="hover:text-white">
                      <X className="w-3 h-3" />
                    </button>
                  </span>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-slate-700 flex items-center justify-between">
          <span className="text-xs text-slate-500">
            {navigator.platform.includes('Mac') ? 'Cmd' : 'Ctrl'}+Enter to save
          </span>
          <div className="flex gap-2">
            <button
              onClick={onClose}
              className="px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-sm transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleSubmit}
              disabled={saving || !content.trim()}
              className="px-4 py-2 bg-primary-500 hover:bg-primary-600 disabled:opacity-50 rounded-lg text-sm transition-colors"
            >
              {saving ? 'Saving...' : 'Save Memory'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

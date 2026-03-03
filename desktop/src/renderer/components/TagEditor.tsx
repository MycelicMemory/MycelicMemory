import { useState } from 'react';
import { Plus, Tag } from 'lucide-react';
import { Badge } from './Badge';

interface TagEditorProps {
  tags: string[];
  onChange: (tags: string[]) => void;
  placeholder?: string;
  readonly?: boolean;
}

export function TagEditor({ tags, onChange, placeholder = 'Add tag...', readonly = false }: TagEditorProps) {
  const [tagInput, setTagInput] = useState('');

  const addTag = () => {
    const trimmed = tagInput.trim().toLowerCase();
    if (trimmed && !tags.includes(trimmed)) {
      onChange([...tags, trimmed]);
    }
    setTagInput('');
  };

  const removeTag = (tag: string) => {
    onChange(tags.filter((t) => t !== tag));
  };

  return (
    <div>
      <label className="text-xs text-slate-400 uppercase tracking-wide flex items-center gap-1">
        <Tag className="w-3 h-3" /> Tags
      </label>
      {!readonly && (
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
            placeholder={placeholder}
            className="flex-1 p-2 bg-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
          <button
            onClick={addTag}
            className="p-2 bg-slate-700 hover:bg-slate-600 rounded-lg transition-colors"
          >
            <Plus className="w-4 h-4" />
          </button>
        </div>
      )}
      {tags.length > 0 && (
        <div className="flex flex-wrap gap-2 mt-2">
          {tags.map((tag) => (
            <Badge key={tag} color="primary" onRemove={readonly ? undefined : () => removeTag(tag)}>
              {tag}
            </Badge>
          ))}
        </div>
      )}
    </div>
  );
}

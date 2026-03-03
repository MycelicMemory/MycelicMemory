import { X } from 'lucide-react';

interface BadgeProps {
  children: React.ReactNode;
  color?: 'primary' | 'green' | 'amber' | 'red' | 'slate';
  size?: 'sm' | 'md';
  onRemove?: () => void;
}

const colorMap = {
  primary: 'bg-primary-500/20 text-primary-400',
  green: 'bg-green-500/20 text-green-400',
  amber: 'bg-amber-500/20 text-amber-400',
  red: 'bg-red-500/20 text-red-400',
  slate: 'bg-slate-700 text-slate-300',
};

export function Badge({ children, color = 'primary', size = 'sm', onRemove }: BadgeProps) {
  const sizeClass = size === 'sm' ? 'text-xs px-2 py-1' : 'text-sm px-3 py-1.5';

  return (
    <span className={`${colorMap[color]} ${sizeClass} rounded inline-flex items-center gap-1`}>
      {children}
      {onRemove && (
        <button onClick={onRemove} className="hover:text-white">
          <X className="w-3 h-3" />
        </button>
      )}
    </span>
  );
}

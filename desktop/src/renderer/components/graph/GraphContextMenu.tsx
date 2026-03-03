import { useEffect, useRef } from 'react';
import { Focus, EyeOff, Pin, PinOff, Copy, Hash, Eye, RotateCcw, Maximize2, Link } from 'lucide-react';

export type ContextMenuAction =
  | 'focus-neighborhood'
  | 'show-only-connected'
  | 'hide-node'
  | 'pin-to-view'
  | 'unpin-from-view'
  | 'copy-content'
  | 'copy-id'
  | 'show-all-hidden'
  | 'reset-view'
  | 'fit-all';

interface GraphContextMenuProps {
  x: number;
  y: number;
  nodeId: string | null;
  nodeType: 'memory' | 'session' | null;
  isPinned: boolean;
  hiddenCount: number;
  onAction: (action: ContextMenuAction) => void;
  onClose: () => void;
}

export default function GraphContextMenu({
  x, y, nodeId, nodeType, isPinned, hiddenCount, onAction, onClose,
}: GraphContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose();
      }
    }
    function handleEscape(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [onClose]);

  // Clamp position so menu doesn't go off-screen
  const clampedX = Math.min(x, window.innerWidth - 220);
  const clampedY = Math.min(y, window.innerHeight - 300);

  const isNode = nodeId !== null;
  const isMemory = nodeType === 'memory';

  function item(label: string, action: ContextMenuAction, icon: React.ReactNode) {
    return (
      <button
        onClick={() => { onAction(action); onClose(); }}
        className="w-full flex items-center gap-2 px-3 py-1.5 text-sm text-slate-200 hover:bg-slate-600 rounded transition-colors text-left"
      >
        {icon}
        {label}
      </button>
    );
  }

  return (
    <div
      ref={menuRef}
      style={{ left: clampedX, top: clampedY }}
      className="fixed z-50 w-52 bg-slate-700 border border-slate-600 rounded-lg shadow-xl py-1"
    >
      {isNode ? (
        <>
          {item('Focus Neighborhood', 'focus-neighborhood', <Focus className="w-3.5 h-3.5" />)}
          {item('Show Only Connected', 'show-only-connected', <Link className="w-3.5 h-3.5" />)}
          {item('Hide Node', 'hide-node', <EyeOff className="w-3.5 h-3.5" />)}
          {isMemory && (
            isPinned
              ? item('Unpin from View', 'unpin-from-view', <PinOff className="w-3.5 h-3.5" />)
              : item('Pin to View', 'pin-to-view', <Pin className="w-3.5 h-3.5" />)
          )}
          {isMemory && item('Copy Content', 'copy-content', <Copy className="w-3.5 h-3.5" />)}
          {item('Copy ID', 'copy-id', <Hash className="w-3.5 h-3.5" />)}
        </>
      ) : (
        <>
          {hiddenCount > 0 && item(`Show All Hidden (${hiddenCount})`, 'show-all-hidden', <Eye className="w-3.5 h-3.5" />)}
          {item('Reset View', 'reset-view', <RotateCcw className="w-3.5 h-3.5" />)}
          {item('Fit All', 'fit-all', <Maximize2 className="w-3.5 h-3.5" />)}
        </>
      )}
    </div>
  );
}

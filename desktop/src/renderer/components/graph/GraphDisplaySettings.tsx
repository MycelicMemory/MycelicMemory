import { ChevronDown, ChevronUp, RotateCcw, X } from 'lucide-react';
import { useState } from 'react';
import type { GraphPhysicsSettings, GraphStyleSettings } from '../../../shared/types';

interface GraphDisplaySettingsProps {
  physics: GraphPhysicsSettings;
  style: GraphStyleSettings;
  onPhysicsChange: (physics: GraphPhysicsSettings) => void;
  onStyleChange: (style: GraphStyleSettings) => void;
  onReset: () => void;
  onClose: () => void;
}

const SMOOTH_TYPES = ['dynamic', 'continuous', 'discrete', 'curvedCW', 'curvedCCW'];

export default function GraphDisplaySettings({
  physics, style, onPhysicsChange, onStyleChange, onReset, onClose,
}: GraphDisplaySettingsProps) {
  const [physicsOpen, setPhysicsOpen] = useState(true);
  const [styleOpen, setStyleOpen] = useState(true);

  function slider(
    label: string,
    value: number,
    min: number,
    max: number,
    step: number,
    onChange: (v: number) => void
  ) {
    return (
      <div className="space-y-1">
        <div className="flex justify-between text-xs">
          <span className="text-slate-400">{label}</span>
          <span className="text-slate-300 font-mono">{value}</span>
        </div>
        <input
          type="range"
          min={min}
          max={max}
          step={step}
          value={value}
          onChange={(e) => onChange(parseFloat(e.target.value))}
          className="w-full h-1 bg-slate-600 rounded-lg appearance-none cursor-pointer accent-indigo-500"
        />
      </div>
    );
  }

  return (
    <div className="absolute top-2 right-2 w-72 z-40 bg-slate-800 border border-slate-600 rounded-lg shadow-xl overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-slate-700">
        <span className="text-sm font-medium">Display Settings</span>
        <button onClick={onClose} className="p-1 hover:bg-slate-700 rounded">
          <X className="w-3.5 h-3.5" />
        </button>
      </div>

      <div className="max-h-[60vh] overflow-y-auto">
        {/* Physics Section */}
        <div className="border-b border-slate-700">
          <button
            onClick={() => setPhysicsOpen(!physicsOpen)}
            className="w-full flex items-center justify-between px-3 py-2 text-xs uppercase text-slate-400 hover:bg-slate-700/50"
          >
            Physics
            {physicsOpen ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
          </button>
          {physicsOpen && (
            <div className="px-3 pb-3 space-y-3">
              {slider('Gravitational Constant', physics.gravitationalConstant, -200, -10, 5, (v) =>
                onPhysicsChange({ ...physics, gravitationalConstant: v })
              )}
              {slider('Spring Length', physics.springLength, 50, 500, 10, (v) =>
                onPhysicsChange({ ...physics, springLength: v })
              )}
              {slider('Spring Constant', physics.springConstant, 0.01, 0.5, 0.01, (v) =>
                onPhysicsChange({ ...physics, springConstant: v })
              )}
              {slider('Damping', physics.damping, 0.1, 0.9, 0.05, (v) =>
                onPhysicsChange({ ...physics, damping: v })
              )}
              {slider('Avoid Overlap', physics.avoidOverlap, 0, 1, 0.1, (v) =>
                onPhysicsChange({ ...physics, avoidOverlap: v })
              )}
            </div>
          )}
        </div>

        {/* Style Section */}
        <div>
          <button
            onClick={() => setStyleOpen(!styleOpen)}
            className="w-full flex items-center justify-between px-3 py-2 text-xs uppercase text-slate-400 hover:bg-slate-700/50"
          >
            Styling
            {styleOpen ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
          </button>
          {styleOpen && (
            <div className="px-3 pb-3 space-y-3">
              {slider('Node Font Size', style.nodeFontSize, 6, 24, 1, (v) =>
                onStyleChange({ ...style, nodeFontSize: v })
              )}
              {slider('Node Border Width', style.nodeBorderWidth, 1, 6, 1, (v) =>
                onStyleChange({ ...style, nodeBorderWidth: v })
              )}
              <div className="space-y-1">
                <span className="text-xs text-slate-400">Edge Smooth Type</span>
                <select
                  value={style.edgeSmoothType}
                  onChange={(e) => onStyleChange({ ...style, edgeSmoothType: e.target.value })}
                  className="w-full px-2 py-1 bg-slate-700 border border-slate-600 rounded text-sm"
                >
                  {SMOOTH_TYPES.map((t) => (
                    <option key={t} value={t}>{t}</option>
                  ))}
                </select>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Footer */}
      <div className="px-3 py-2 border-t border-slate-700">
        <button
          onClick={onReset}
          className="w-full flex items-center justify-center gap-1.5 px-3 py-1.5 text-xs text-slate-300 bg-slate-700 hover:bg-slate-600 rounded transition-colors"
        >
          <RotateCcw className="w-3 h-3" />
          Reset to Defaults
        </button>
      </div>
    </div>
  );
}

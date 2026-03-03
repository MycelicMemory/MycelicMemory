import { useState, useCallback, useRef, useEffect } from 'react';
import type { GraphPhysicsSettings, GraphStyleSettings } from '../../shared/types';

export const DEFAULT_PHYSICS: GraphPhysicsSettings = {
  gravitationalConstant: -80,
  centralGravity: 0.005,
  springLength: 230,
  springConstant: 0.08,
  damping: 0.4,
  avoidOverlap: 0.8,
  maxVelocity: 30,
  timestep: 0.35,
};

export const DEFAULT_STYLE: GraphStyleSettings = {
  nodeFontSize: 12,
  nodeBorderWidth: 2,
  edgeFontSize: 10,
  edgeSmoothType: 'dynamic',
};

export function useGraphSettings(networkRef: React.MutableRefObject<any>) {
  const [physics, setPhysics] = useState<GraphPhysicsSettings>({ ...DEFAULT_PHYSICS });
  const [style, setStyle] = useState<GraphStyleSettings>({ ...DEFAULT_STYLE });
  const [tuning, setTuning] = useState(false);

  // Expose tuning flag via ref so event handlers (registered once) can read it
  const tuningRef = useRef(false);
  useEffect(() => { tuningRef.current = tuning; }, [tuning]);

  const applyPhysics = useCallback((next: GraphPhysicsSettings) => {
    setPhysics(next);
    if (!networkRef.current) return;

    // Apply new physics options — keep physics enabled, don't call stabilize()
    // The network will continuously simulate with the new parameters
    networkRef.current.setOptions({
      physics: {
        enabled: true,
        forceAtlas2Based: {
          gravitationalConstant: next.gravitationalConstant,
          centralGravity: next.centralGravity,
          springLength: next.springLength,
          springConstant: next.springConstant,
          damping: next.damping,
          avoidOverlap: next.avoidOverlap,
        },
        maxVelocity: next.maxVelocity,
        timestep: next.timestep,
        stabilization: false,
      },
    });
  }, [networkRef]);

  const applyStyle = useCallback((next: GraphStyleSettings) => {
    setStyle(next);
    if (!networkRef.current) return;

    networkRef.current.setOptions({
      nodes: {
        font: { size: next.nodeFontSize },
        borderWidth: next.nodeBorderWidth,
      },
      edges: {
        font: { size: next.edgeFontSize },
        smooth: { type: next.edgeSmoothType },
      },
    });
  }, [networkRef]);

  const resetToDefaults = useCallback(() => {
    applyPhysics({ ...DEFAULT_PHYSICS });
    applyStyle({ ...DEFAULT_STYLE });
  }, [applyPhysics, applyStyle]);

  // When tuning stops (settings panel closes), freeze physics after a settle period
  useEffect(() => {
    if (!tuning && networkRef.current) {
      const timer = setTimeout(() => {
        if (!tuningRef.current && networkRef.current) {
          networkRef.current.setOptions({ physics: { enabled: false } });
        }
      }, 2000);
      return () => clearTimeout(timer);
    }
  }, [tuning, networkRef]);

  return { physics, style, tuning, setTuning, tuningRef, applyPhysics, applyStyle, resetToDefaults };
}

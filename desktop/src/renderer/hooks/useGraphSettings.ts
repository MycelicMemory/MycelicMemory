import { useState, useCallback, useRef } from 'react';
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

  const applyPhysics = useCallback((next: GraphPhysicsSettings) => {
    setPhysics(next);
    if (networkRef.current) {
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
        },
      });
      networkRef.current.stabilize(100);
    }
  }, [networkRef]);

  const applyStyle = useCallback((next: GraphStyleSettings) => {
    setStyle(next);
    if (networkRef.current) {
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
    }
  }, [networkRef]);

  const resetToDefaults = useCallback(() => {
    applyPhysics({ ...DEFAULT_PHYSICS });
    applyStyle({ ...DEFAULT_STYLE });
  }, [applyPhysics, applyStyle]);

  return { physics, style, applyPhysics, applyStyle, resetToDefaults };
}

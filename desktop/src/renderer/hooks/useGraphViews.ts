import { useState, useEffect, useCallback } from 'react';
import type { GraphView } from '../../shared/types';

export function useGraphViews() {
  const [views, setViews] = useState<GraphView[]>([]);
  const [activeViewId, setActiveViewId] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try {
      const list = await window.mycelicMemory.graphViews.list();
      setViews(list);
      const active = await window.mycelicMemory.graphViews.getActive();
      setActiveViewId(active?.id);
    } catch (err) {
      console.error('Failed to load graph views:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const saveView = useCallback(async (view: GraphView) => {
    const saved = await window.mycelicMemory.graphViews.save(view);
    await window.mycelicMemory.graphViews.setActive(saved.id);
    await refresh();
    return saved;
  }, [refresh]);

  const deleteView = useCallback(async (id: string) => {
    await window.mycelicMemory.graphViews.delete(id);
    await refresh();
  }, [refresh]);

  return { views, activeViewId, loading, refresh, saveView, deleteView };
}

/**
 * Graph Views IPC Handlers
 * Manages saved graph view configurations (filters, physics, style, hidden/pinned nodes)
 */

import type { IpcMain } from 'electron';
import type Store from 'electron-store';
import type { AppSettings, GraphView, GraphViewStore } from '../../shared/types';

export function registerGraphViewsHandlers(
  ipcMain: IpcMain,
  store: Store<{ settings: AppSettings; graphViews: GraphViewStore }>
): void {
  ipcMain.handle('graph-views:list', () => {
    return store.get('graphViews.views', []);
  });

  ipcMain.handle('graph-views:get', (_event, params: { id: string }) => {
    const views = store.get('graphViews.views', []) as GraphView[];
    return views.find((v) => v.id === params.id) ?? null;
  });

  ipcMain.handle('graph-views:save', (_event, view: GraphView) => {
    const views = store.get('graphViews.views', []) as GraphView[];
    const idx = views.findIndex((v) => v.id === view.id);
    view.updated_at = new Date().toISOString();
    if (idx >= 0) {
      views[idx] = view;
    } else {
      views.push(view);
    }
    store.set('graphViews.views', views);
    return view;
  });

  ipcMain.handle('graph-views:delete', (_event, params: { id: string }) => {
    const views = store.get('graphViews.views', []) as GraphView[];
    const filtered = views.filter((v) => v.id !== params.id);
    store.set('graphViews.views', filtered);
    const activeId = store.get('graphViews.activeViewId') as string | undefined;
    if (activeId === params.id) {
      store.set('graphViews.activeViewId', undefined);
    }
    return filtered.length < views.length;
  });

  ipcMain.handle('graph-views:set-active', (_event, params: { id: string | null }) => {
    store.set('graphViews.activeViewId', params.id ?? undefined);
    return true;
  });

  ipcMain.handle('graph-views:get-active', () => {
    const activeId = store.get('graphViews.activeViewId') as string | undefined;
    if (!activeId) return null;
    const views = store.get('graphViews.views', []) as GraphView[];
    return views.find((v) => v.id === activeId) ?? null;
  });
}

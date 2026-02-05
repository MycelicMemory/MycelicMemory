import { ipcMain } from 'electron';
import { MycelicMemoryClient } from '../services/mycelicmemory-client';
import type {
  DataSource,
  DataSourceCreateInput,
  DataSourceUpdateInput,
  DataSourceStats,
  SyncHistoryEntry,
  IngestRequest,
  IngestResponse,
  Memory,
  DataSourceType,
  DataSourceStatus,
} from '../../shared/types';

/**
 * Initialize IPC handlers for Data Source operations
 */
export function initSourcesIPC(client: MycelicMemoryClient): void {
  // List all data sources
  ipcMain.handle('sources:list', async (_event, params: { source_type?: DataSourceType; status?: DataSourceStatus }) => {
    const queryParams = new URLSearchParams();
    if (params.source_type) queryParams.set('source_type', params.source_type);
    if (params.status) queryParams.set('status', params.status);

    const query = queryParams.toString();
    const endpoint = query ? `/sources?${query}` : '/sources';

    const response = await client.get<{ data: DataSource[] }>(endpoint);
    return response.data || [];
  });

  // Get a single data source
  ipcMain.handle('sources:get', async (_event, params: { id: string }) => {
    try {
      const response = await client.get<{ data: DataSource }>(`/sources/${params.id}`);
      return response.data || null;
    } catch (error: any) {
      if (error.message?.includes('404') || error.message?.includes('not found')) {
        return null;
      }
      throw error;
    }
  });

  // Create a new data source
  ipcMain.handle('sources:create', async (_event, params: DataSourceCreateInput) => {
    const response = await client.post<{ data: DataSource }>('/sources', params);
    return response.data;
  });

  // Update a data source
  ipcMain.handle('sources:update', async (_event, params: { id: string; data: DataSourceUpdateInput }) => {
    const response = await client.patch<{ data: DataSource }>(`/sources/${params.id}`, params.data);
    return response.data;
  });

  // Delete a data source
  ipcMain.handle('sources:delete', async (_event, params: { id: string }) => {
    await client.delete(`/sources/${params.id}`);
    return true;
  });

  // Pause a data source
  ipcMain.handle('sources:pause', async (_event, params: { id: string }) => {
    const response = await client.post<{ data: DataSource }>(`/sources/${params.id}/pause`, {});
    return response.data;
  });

  // Resume a data source
  ipcMain.handle('sources:resume', async (_event, params: { id: string }) => {
    const response = await client.post<{ data: DataSource }>(`/sources/${params.id}/resume`, {});
    return response.data;
  });

  // Trigger sync for a data source
  ipcMain.handle('sources:sync', async (_event, params: { id: string }) => {
    const response = await client.post<{ data: SyncHistoryEntry }>(`/sources/${params.id}/sync`, {});
    return response.data;
  });

  // Ingest items into a data source
  ipcMain.handle('sources:ingest', async (_event, params: { id: string; request: IngestRequest }) => {
    const response = await client.post<{ data: IngestResponse }>(`/sources/${params.id}/ingest`, params.request);
    return response.data;
  });

  // Get sync history for a data source
  ipcMain.handle('sources:history', async (_event, params: { id: string; limit?: number }) => {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.set('limit', params.limit.toString());

    const query = queryParams.toString();
    const endpoint = query ? `/sources/${params.id}/history?${query}` : `/sources/${params.id}/history`;

    const response = await client.get<{ data: SyncHistoryEntry[] }>(endpoint);
    return response.data || [];
  });

  // Get stats for a data source
  ipcMain.handle('sources:stats', async (_event, params: { id: string }) => {
    const response = await client.get<{ data: DataSourceStats }>(`/sources/${params.id}/stats`);
    return response.data;
  });

  // Get memories for a data source
  ipcMain.handle('sources:memories', async (_event, params: { id: string; limit?: number; offset?: number }) => {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.set('limit', params.limit.toString());
    if (params.offset) queryParams.set('offset', params.offset.toString());

    const query = queryParams.toString();
    const endpoint = query ? `/sources/${params.id}/memories?${query}` : `/sources/${params.id}/memories`;

    const response = await client.get<{ data: Memory[] }>(endpoint);
    return response.data || [];
  });
}

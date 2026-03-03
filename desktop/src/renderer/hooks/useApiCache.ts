import { useState, useEffect, useCallback, useRef } from 'react';

interface CacheEntry<T> {
  data: T;
  timestamp: number;
}

const cache = new Map<string, CacheEntry<any>>();

/**
 * Invalidate all cache entries matching a key prefix.
 * Call after mutations (create/update/delete).
 */
export function invalidateCache(keyPrefix?: string) {
  if (!keyPrefix) {
    cache.clear();
    return;
  }
  for (const key of cache.keys()) {
    if (key.startsWith(keyPrefix)) {
      cache.delete(key);
    }
  }
}

/**
 * Hook for caching API responses with TTL.
 * Returns cached data immediately if fresh, fetches in background if stale.
 */
export function useApiCache<T>(
  key: string,
  fetcher: () => Promise<T>,
  ttlMs: number
): {
  data: T | null;
  loading: boolean;
  error: Error | null;
  refresh: () => void;
} {
  const [data, setData] = useState<T | null>(() => {
    const entry = cache.get(key);
    if (entry && Date.now() - entry.timestamp < ttlMs) {
      return entry.data;
    }
    return null;
  });
  const [loading, setLoading] = useState(!data);
  const [error, setError] = useState<Error | null>(null);
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;

  const doFetch = useCallback(async (force = false) => {
    // Check cache first (unless forced)
    if (!force) {
      const entry = cache.get(key);
      if (entry && Date.now() - entry.timestamp < ttlMs) {
        setData(entry.data);
        setLoading(false);
        return;
      }
    }

    setLoading(true);
    setError(null);
    try {
      const result = await fetcherRef.current();
      cache.set(key, { data: result, timestamp: Date.now() });
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoading(false);
    }
  }, [key, ttlMs]);

  useEffect(() => {
    doFetch();
  }, [doFetch]);

  const refresh = useCallback(() => {
    cache.delete(key);
    doFetch(true);
  }, [key, doFetch]);

  return { data, loading, error, refresh };
}

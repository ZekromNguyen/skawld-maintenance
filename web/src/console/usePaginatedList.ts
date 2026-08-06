import { useCallback, useEffect, useRef, useState } from "react";
import type { ListPage } from "../types";

export interface PaginatedResult<T> {
  items: T[];
  loading: boolean;
  error: string | undefined;
  hasMore: boolean;
  loadMore: () => Promise<void>;
  refetch: () => Promise<void>;
}

/**
 * usePaginatedList: cursor-walking list state. Loads the first page,
 * appends subsequent pages on loadMore, and refetches from the start.
 */
export function usePaginatedList<T>(
  fetcher: (params: { page_size: number; cursor?: string }) => Promise<ListPage<T>>,
  deps: unknown[],
  pageSize = 25,
): PaginatedResult<T> {
  const [items, setItems] = useState<T[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>(undefined);
  const [hasMore, setHasMore] = useState(false);
  const cursorRef = useRef<string | undefined>(undefined);
  const seqRef = useRef(0);

  const run = useCallback(async (fromStart: boolean) => {
    const seq = ++seqRef.current;
    setLoading(true);
    setError(undefined);
    try {
      const cursor = fromStart ? undefined : cursorRef.current;
      const page = await fetcher({ page_size: pageSize, cursor });
      if (seq !== seqRef.current) return;
      cursorRef.current = page.next_cursor ?? undefined;
      setHasMore(page.has_more);
      setItems((previous) => (fromStart ? page.items : [...previous, ...page.items]));
    } catch (err) {
      if (seq !== seqRef.current) return;
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      if (seq === seqRef.current) setLoading(false);
    }
  }, deps); // eslint-disable-line react-hooks/exhaustive-deps

  const loadMore = useCallback(async () => { await run(false); }, [run]);
  const refetch = useCallback(async () => { await run(true); }, [run]);

  useEffect(() => { void run(true); }, [run]);

  return { items, loading, error, hasMore, loadMore, refetch };
}

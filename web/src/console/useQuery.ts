import { useCallback, useEffect, useRef, useState } from "react";

export interface QueryResult<T> {
  data: T | undefined;
  loading: boolean;
  error: string | undefined;
  refetch: () => Promise<T | undefined>;
}

export function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

/**
 * useQuery: fetch with dependency-driven refetch, stale-response dropping,
 * and an awaitable refetch. Replaces useApi (which could not re-run when the
 * site/principal resolved and silently raced overlapping requests).
 */
export function useQuery<T>(
  fetcher: () => Promise<T>,
  deps: unknown[] = [],
): QueryResult<T> {
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;
  const seqRef = useRef(0);
  const [data, setData] = useState<T | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>(undefined);

  const run = useCallback(async (): Promise<T | undefined> => {
    const seq = ++seqRef.current;
    setLoading(true);
    setError(undefined);
    try {
      const value = await fetcherRef.current();
      if (seq !== seqRef.current) return undefined;
      setData(value);
      return value;
    } catch (err) {
      if (seq !== seqRef.current) return undefined;
      setError(errorMessage(err));
      return undefined;
    } finally {
      if (seq === seqRef.current) setLoading(false);
    }
  }, []);

  useEffect(() => {
    void run();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [run, ...deps]);

  return { data, loading, error, refetch: run };
}

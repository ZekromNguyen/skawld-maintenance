import { useCallback, useEffect, useState } from "react";

/**
 * useApi: fetch wrapper with data/loading/error + refetch. The fetcher must
 * be stable (wrap in useCallback) to avoid refetch loops.
 */
export function useApi<T>(fetcher: () => Promise<T>) {
  const [data, setData] = useState<T | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>(undefined);

  const load = useCallback(() => {
    let cancelled = false;
    setLoading(true);
    setError(undefined);
    fetcher()
      .then((value) => {
        if (!cancelled) setData(value);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [fetcher]);

  useEffect(() => load(), [load]);

  return { data, loading, error, refetch: load };
}

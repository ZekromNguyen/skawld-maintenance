import { useCallback, useRef, useState } from "react";
import { errorMessage } from "./useQuery";

export interface MutationResult<TArgs extends unknown[], TResult> {
  pending: boolean;
  error: string | undefined;
  run: (...args: TArgs) => Promise<TResult | undefined>;
}

/**
 * useMutation: executes an async action, capturing failures instead of
 * throwing. Error text is exposed for toast wiring (see useCommand).
 */
export function useMutation<TArgs extends unknown[], TResult>(
  action: (...args: TArgs) => Promise<TResult>,
): MutationResult<TArgs, TResult> {
  const actionRef = useRef(action);
  actionRef.current = action;
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | undefined>(undefined);

  const run = useCallback(async (...args: TArgs): Promise<TResult | undefined> => {
    setPending(true);
    setError(undefined);
    try {
      return await actionRef.current(...args);
    } catch (err) {
      setError(errorMessage(err));
      return undefined;
    } finally {
      setPending(false);
    }
  }, []);

  return { pending, error, run };
}

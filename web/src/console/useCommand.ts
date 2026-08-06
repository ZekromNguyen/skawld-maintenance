import { useCallback, useRef, useState } from "react";
import { errorMessage } from "./useQuery";
import { useToast } from "./feedback/Toast";

export interface CommandResult<TArgs extends unknown[], TResult> {
  pending: boolean;
  run: (...args: TArgs) => Promise<TResult | undefined>;
}

/**
 * useCommand: useMutation + automatic toast feedback. Errors always toast
 * (with optional retry callback); success toasts only when successMessage is
 * provided. Pages stop writing try/catch/void boilerplate.
 */
export function useCommand<TArgs extends unknown[], TResult>(
  action: (...args: TArgs) => Promise<TResult>,
  options?: {
    successMessage?: string;
    onSuccess?: (result: TResult) => void;
    onError?: (message: string) => void;
    retry?: (...args: TArgs) => void;
  },
): CommandResult<TArgs, TResult> {
  const actionRef = useRef(action);
  actionRef.current = action;
  const optionsRef = useRef(options);
  optionsRef.current = options;
  const toast = useToast();
  const [pending, setPending] = useState(false);

  const run = useCallback(
    async (...args: TArgs): Promise<TResult | undefined> => {
      setPending(true);
      try {
        const result = await actionRef.current(...args);
        if (optionsRef.current?.successMessage) toast.success(optionsRef.current.successMessage);
        optionsRef.current?.onSuccess?.(result);
        return result;
      } catch (err) {
        const message = errorMessage(err);
        toast.error(message, {
          retry: optionsRef.current?.retry
            ? () => optionsRef.current!.retry!(...args)
            : undefined,
        });
        optionsRef.current?.onError?.(message);
        return undefined;
      } finally {
        setPending(false);
      }
    },
    [toast],
  );

  return { pending, run };
}

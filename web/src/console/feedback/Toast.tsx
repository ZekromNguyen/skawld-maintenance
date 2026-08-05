import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from "react";

type ToastKind = "success" | "error" | "info";
interface ToastItem {
  id: number;
  kind: ToastKind;
  message: string;
  retry?: () => void;
}
interface ToastApi {
  success: (message: string) => void;
  error: (message: string, opts?: { retry?: () => void }) => void;
  info: (message: string) => void;
}

const ToastContext = createContext<ToastApi | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const nextId = useRef(1);

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  }, []);

  const push = useCallback(
    (kind: ToastKind, message: string, opts?: { retry?: () => void }) => {
      const id = nextId.current++;
      setToasts((prev) => [...prev.slice(-3), { id, kind, message, retry: opts?.retry }]);
      if (kind !== "error") {
        setTimeout(() => dismiss(id), 5000);
      }
    },
    [dismiss],
  );

  const api: ToastApi = {
    success: (message) => push("success", message),
    error: (message, opts) => push("error", message, opts),
    info: (message) => push("info", message),
  };

  return (
    <ToastContext.Provider value={api}>
      {children}
      <div className="toast-region" role="region" aria-label="Notifications">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            role={toast.kind === "error" ? "alert" : "status"}
            className={`toast toast-${toast.kind}`}
          >
            <span>{toast.message}</span>
            {toast.retry ? (
              <button
                className="toast-retry"
                onClick={() => {
                  toast.retry?.();
                  dismiss(toast.id);
                }}
              >
                Retry
              </button>
            ) : null}
            <button
              className="toast-close"
              aria-label="Dismiss"
              onClick={() => dismiss(toast.id)}
            >
              ×
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastApi {
  const value = useContext(ToastContext);
  if (!value) throw new Error("useToast must be used within ToastProvider");
  return value;
}

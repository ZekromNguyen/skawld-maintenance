# Skawld Console Redesign Implementation Plan (Phase 0 + Phase 1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the console's foundation (data layer, feedback/display primitives, shell) and redesign the operations pages (Dashboard, Incidents, Executions, Workbench) so every flow is reachable, every mutation gives feedback, multi-site scope is correct, and the UI speaks one instrument-state language.

**Architecture:** A shared foundation under `web/src/console/` — `useQuery`/`useMutation` replace the swallow-errors `useApi`; `SiteContext` + `PrincipalProvider` fix scope and duplicate `/me`; Radix `Dialog`/`ConfirmDialog` + `ToastProvider` give every action feedback; `DataTable`/`StatusBadge`/`EmptyState`/`ErrorState`/`FormField`/`RelativeTime`/`MetricCard`/`Skeleton` standardize rendering; a `PageHeader` + trail context fixes the dead breadcrumb prop. Pages are rewritten one at a time against these primitives. Console stays dark-first; buttons are 8px radius (documented console-surface deviation from the marketing pill); API-dependent features ship with client-side fallbacks.

**Tech Stack:** React 19, react-router 7, Vite, vitest + jsdom + Testing Library, @radix-ui/react-dialog, @phosphor-icons/react, @fontsource-variable/geist.

## Global Constraints

- **Fonts:** console must use Geist (Inter is banned). `@fontsource-variable/geist` is already loaded globally in `main.tsx`; change the `font-family` stack in `styles.css`.
- **Shape lock (console surface):** cards 12px, inputs 10px, buttons 8px, dialogs 16px. Pill buttons stay marketing-only.
- **Dark-first only** for the console; no light-mode work.
- **Tone language:** severity/state must use the `.tone-*` utilities; no raw enum strings, no `toLowerCase()` class hacks, no all-green status dots.
- **Feedback:** no silent mutations — every mutation ends in a toast (success or error). Destructive/safety actions (resolve, LOTO step complete, retire) require `ConfirmDialog`.
- **i18n:** every new visible string needs EN + VI keys in `messages.ts` (`MessageKey = keyof typeof en` enforces VI parity).
- **Site scope:** fetches depend on `useSite().siteId`; no `site_ids[0]` pinning outside the SiteProvider default.
- **Tests:** TDD per task; every new primitive gets a test; existing suites must stay green. Run `cd web && npm run lint && npm test` after every task.

---

### Task 0.1: Geist font + console tone tokens

**Files:**
- Modify: `web/src/styles.css:3-24` (root font + new tokens)
- Test: `web/src/console/useQuery.test.ts` (placeholder — see Task 0.2) — none; verify visually via build.

- [ ] **Step 1: Swap the font stack and add tone tokens**

In `web/src/styles.css`, replace the `:root` font-family line:

```css
  font-family: "Geist Variable", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
```

Add to `:root` (after `--blue`):

```css
  --radius-card: 12px;
  --radius-input: 10px;
  --radius-button: 8px;
  --radius-dialog: 16px;
  --ring: 0 0 0 2px var(--brand);
  --tone-critical-bg: #3a1412;
  --tone-critical-fg: #ffb3a8;
  --tone-critical-line: #5a211c;
  --tone-high-bg: #3a2414;
  --tone-high-fg: #ffd9b3;
  --tone-high-line: #5c3a20;
  --tone-medium-bg: #332b14;
  --tone-medium-fg: #f3e0a8;
  --tone-medium-line: #554a22;
  --tone-low-bg: #143024;
  --tone-low-fg: #b3e8cd;
  --tone-low-line: #22503c;
  --tone-info-fg: #7fb3e0;
  --tone-info-bg: #14263a;
  --tone-info-line: #22445e;
  --tone-success-fg: #b3e8cd;
  --tone-success-bg: #143024;
  --tone-success-line: #22503c;
```

- [ ] **Step 2: Add tone utilities and global focus-visible**

Append to `web/src/styles.css`:

```css
/* ---------- Tone utilities (console state language) ---------- */
.tone {
  display: inline-flex;
  align-items: center;
  min-height: 20px;
  padding: 2px 7px;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.04em;
  border: 1px solid transparent;
}
.tone-critical { color: var(--tone-critical-fg); background: var(--tone-critical-bg); border-color: var(--tone-critical-line); }
.tone-high { color: var(--tone-high-fg); background: var(--tone-high-bg); border-color: var(--tone-high-line); }
.tone-medium { color: var(--tone-medium-fg); background: var(--tone-medium-bg); border-color: var(--tone-medium-line); }
.tone-low { color: var(--tone-low-fg); background: var(--tone-low-bg); border-color: var(--tone-low-line); }
.tone-info { color: var(--tone-info-fg); background: var(--tone-info-bg); border-color: var(--tone-info-line); }
.tone-success { color: var(--tone-success-fg); background: var(--tone-success-bg); border-color: var(--tone-success-line); }

:focus-visible {
  outline: 2px solid var(--brand);
  outline-offset: 2px;
}
button:focus:not(:focus-visible),
a:focus:not(:focus-visible) {
  outline: none;
}
```

- [ ] **Step 3: Verify build**

Run: `cd web && npm run build`
Expected: succeeds.

- [ ] **Step 4: Commit**

```bash
git add web/src/styles.css
git commit -m "feat(web): console tone system and Geist font swap"
```

---

### Task 0.2: Data layer — useQuery, useMutation, useCommand

**Files:**
- Create: `web/src/console/useQuery.ts`
- Create: `web/src/console/useMutation.ts`
- Modify: `web/src/console/usePrincipal.ts` (backed by provider — Task 0.6 wires the provider; keep the hook's return shape `{ data: principal }` so call sites compile)
- Test: `web/src/console/useQuery.test.ts`, `web/src/console/useMutation.test.ts`

**Interfaces:**
- Produces:
  - `useQuery<T>(fetcher: () => Promise<T>, deps?: unknown[]): { data: T | undefined; loading: boolean; error: string | undefined; refetch: () => Promise<T | undefined> }`
  - `useMutation<TArgs extends unknown[], TResult>(action: (...args: TArgs) => Promise<TResult>): { pending: boolean; error: string | undefined; run: (...args: TArgs) => Promise<TResult | undefined> }`
  - `useCommand<TArgs extends unknown[], TResult>(action, opts?: { successMessage?: string; onSuccess?: (r: TResult) => void; onError?: (msg: string) => void }): { pending; run }` — auto-toasts errors (requires `useToast`, Task 0.4; import from `./feedback/Toast`).

- [ ] **Step 1: Write failing tests**

Create `web/src/console/useQuery.test.ts`:

```ts
import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { useQuery } from "./useQuery";

describe("useQuery", () => {
  it("loads data and exposes it", async () => {
    const fetcher = vi.fn().mockResolvedValue("ok");
    const { result } = renderHook(() => useQuery(fetcher));
    expect(result.current.loading).toBe(true);
    await waitFor(() => expect(result.current.data).toBe("ok"));
    expect(result.current.loading).toBe(false);
    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it("refetches when a dependency changes", async () => {
    const fetcher = vi.fn().mockResolvedValue("a");
    const { result, rerender } = renderHook(({ dep }) => useQuery(fetcher, [dep]), {
      initialProps: { dep: "x" },
    });
    await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(1));
    fetcher.mockResolvedValue("b");
    rerender({ dep: "y" });
    await waitFor(() => expect(result.current.data).toBe("b"));
    expect(fetcher).toHaveBeenCalledTimes(2);
  });

  it("drops stale responses (last request wins)", async () => {
    let resolveFirst: (v: string) => void = () => {};
    const fetcher = vi.fn().mockImplementationOnce(
      () => new Promise<string>((resolve) => { resolveFirst = resolve; }),
    );
    const { result } = renderHook(() => useQuery(fetcher));
    const first = result.current.refetch();
    fetcher.mockResolvedValueOnce("second");
    await result.current.refetch();
    await waitFor(() => expect(result.current.data).toBe("second"));
    act(() => resolveFirst("stale"));
    await first;
    expect(result.current.data).toBe("second");
  });

  it("surfaces errors and recovers on refetch", async () => {
    const fetcher = vi.fn().mockRejectedValueOnce(new Error("boom")).mockResolvedValueOnce("ok");
    const { result } = renderHook(() => useQuery(fetcher));
    await waitFor(() => expect(result.current.error).toBe("boom"));
    await act(async () => { await result.current.refetch(); });
    expect(result.current.data).toBe("ok");
    expect(result.current.error).toBeUndefined();
  });
});
```

Create `web/src/console/useMutation.test.ts`:

```ts
import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { useMutation } from "./useMutation";

describe("useMutation", () => {
  it("runs the action and reports pending", async () => {
    const action = vi.fn().mockResolvedValue(42);
    const { result } = renderHook(() => useMutation(action));
    let resolved: number | undefined;
    act(() => { void result.current.run(1).then((v) => { resolved = v; }); });
    expect(result.current.pending).toBe(true);
    await waitFor(() => expect(result.current.pending).toBe(false));
    expect(action).toHaveBeenCalledWith(1);
    expect(resolved).toBe(42);
  });

  it("captures the error instead of throwing", async () => {
    const action = vi.fn().mockRejectedValue(new Error("nope"));
    const { result } = renderHook(() => useMutation(action));
    await act(async () => { await result.current.run(); });
    expect(result.current.error).toBe("nope");
    expect(result.current.pending).toBe(false);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web && npx vitest run src/console/useQuery.test.ts src/console/useMutation.test.ts`
Expected: FAIL (module not found).

- [ ] **Step 3: Implement useQuery**

Create `web/src/console/useQuery.ts`:

```ts
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
```

- [ ] **Step 4: Implement useMutation**

Create `web/src/console/useMutation.ts`:

```ts
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
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd web && npx vitest run src/console/useQuery.test.ts src/console/useMutation.test.ts`
Expected: PASS.

- [ ] **Step 6: Type-check and commit**

Run: `cd web && npm run lint`
Expected: clean.

```bash
git add web/src/console/useQuery.ts web/src/console/useMutation.ts web/src/console/useQuery.test.ts web/src/console/useMutation.test.ts
git commit -m "feat(web): add query and mutation hooks with stale-response guards"
```

---

### Task 0.3: api.ts hardening

**Files:**
- Modify: `web/src/api.ts:29-51`
- Test: `web/src/api.test.ts` (create)

**Interfaces:**
- Produces: `request()` never throws `SyntaxError` on non-JSON bodies; `ApiError` carries `status`; network failures surface as `ApiError` with a friendly message.

- [ ] **Step 1: Write failing tests**

Create `web/src/api.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { api } from "./api";

describe("api request hardening", () => {
  const originalFetch = globalThis.fetch;
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("turns a non-JSON error body into a readable message", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response("<html>gateway error</html>", { status: 502 }),
    );
    await expect(api.incidents()).rejects.toThrow(/502|response|html/i);
  });

  it("exposes the HTTP status on the error", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ detail: "nope" }), { status: 403 }),
    );
    try {
      await api.incidents();
      expect.unreachable();
    } catch (err) {
      expect((err as { status?: number }).status).toBe(403);
    }
  });
});
```

Note: if `api.incidents()` signature makes this awkward in jsdom, test the module's internal `request` via a documented export `request(path, init)`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web && npx vitest run src/api.test.ts`
Expected: FAIL — `SyntaxError` or missing `status`.

- [ ] **Step 3: Harden request()**

In `web/src/api.ts`, replace the request body:

```ts
  let response: Response;
  try {
    response = await fetch(input, init);
  } catch (err) {
    throw new ApiError(0, "Network error: " + (err instanceof Error ? err.message : String(err)));
  }
  if (response.status === 401) {
    window.location.assign("/auth/login");
    throw new ApiError(401, "Session expired");
  }
  let payload: unknown;
  try {
    payload = await response.json();
  } catch {
    payload = null;
  }
  if (!response.ok) {
    const detail =
      payload && typeof payload === "object" && "detail" in payload
        ? String((payload as { detail: unknown }).detail)
        : `Request failed with status ${response.status}`;
    throw new ApiError(response.status, detail);
  }
  return payload as T;
```

Add the `ApiError` class above `request`:

```ts
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}
```

Preserve the existing `credentials`, headers, and `Idempotency-Key` logic; keep `command()` unchanged. Read the file first and edit the exact current block.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web && npx vitest run src/api.test.ts`
Expected: PASS.

- [ ] **Step 5: Lint and commit**

```bash
git add web/src/api.ts web/src/api.test.ts
git commit -m "fix(web): surface HTTP status and readable errors from the api client"
```

---

### Task 0.4: Feedback primitives — ToastProvider, Dialog, ConfirmDialog

**Files:**
- Create: `web/src/console/feedback/Toast.tsx`
- Create: `web/src/console/feedback/Dialog.tsx`
- Create: `web/src/console/feedback/ConfirmDialog.tsx`
- Create: `web/src/console/feedback/Toast.test.tsx`
- Modify: `web/src/styles.css` (toast/dialog styles)

**Interfaces:**
- Produces:
  - `ToastProvider` (mount once in `App` around the router) + `useToast(): { success(msg): void; error(msg, opts?: { retry?: () => void }): void; info(msg): void }`
  - `Dialog({ open, onOpenChange, title, description?, children, footer? })` — Radix-based, focus-trapped, ESC-close.
  - `ConfirmDialog({ open, onOpenChange, title, message, confirmLabel, danger?, onConfirm })` — used by Task 1.3/1.5.

- [ ] **Step 1: Write failing tests**

Create `web/src/console/feedback/Toast.test.tsx`:

```tsx
import { describe, it, expect, vi } from "vitest";
import { act } from "@testing-library/react";
import { render, screen, fireEvent } from "@testing-library/react";
import { ToastProvider, useToast } from "./Toast";

function Probe() {
  const toast = useToast();
  return (
    <button onClick={() => toast.success("Saved")}>fire</button>
  );
}

describe("ToastProvider", () => {
  it("shows a success toast", () => {
    render(
      <ToastProvider>
        <Probe />
      </ToastProvider>,
    );
    fireEvent.click(screen.getByRole("button", { name: "fire" }));
    expect(screen.getByText("Saved")).toBeTruthy();
  });

  it("surfaces an error toast with retry", () => {
    const retry = vi.fn();
    function ProbeRetry() {
      const toast = useToast();
      return <button onClick={() => toast.error("Failed", { retry })}>go</button>;
    }
    render(
      <ToastProvider>
        <ProbeRetry />
      </ToastProvider>,
    );
    fireEvent.click(screen.getByRole("button", { name: "go" }));
    expect(screen.getByText("Failed")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(retry).toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run to verify they fail**

Run: `cd web && npx vitest run src/console/feedback/Toast.test.tsx`
Expected: FAIL (module not found).

- [ ] **Step 3: Implement Toast**

Create `web/src/console/feedback/Toast.tsx`:

```tsx
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
              <button className="toast-retry" onClick={() => { toast.retry?.(); dismiss(toast.id); }}>
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
```

- [ ] **Step 4: Add toast styles to styles.css**

```css
/* ---------- Toasts ---------- */
.toast-region { position: fixed; right: 20px; bottom: 20px; z-index: 80; display: grid; gap: 8px; width: min(360px, calc(100vw - 40px)); }
.toast { display: flex; align-items: flex-start; gap: 10px; padding: 12px 14px; border-radius: 10px; border: 1px solid var(--line-soft); background: var(--surface); color: var(--ink); font-size: 12.5px; box-shadow: 0 12px 40px rgba(0,0,0,.4); }
.toast-success { border-left: 3px solid var(--brand); }
.toast-error { border-left: 3px solid var(--red); }
.toast-info { border-left: 3px solid var(--blue); }
.toast span { flex: 1; line-height: 1.5; }
.toast button { border: 0; background: transparent; color: var(--ink-soft); font-size: 11px; font-weight: 650; padding: 2px 4px; cursor: pointer; }
.toast-retry { color: var(--brand) !important; }
```

- [ ] **Step 5: Implement Dialog and ConfirmDialog**

Create `web/src/console/feedback/Dialog.tsx`:

```tsx
import * as DialogPrimitive from "@radix-ui/react-dialog";
import type { ReactNode } from "react";

export function Dialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  footer,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  children?: ReactNode;
  footer?: ReactNode;
}) {
  return (
    <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="dialog-overlay" />
        <DialogPrimitive.Content className="dialog-content" aria-describedby={description ? undefined : undefined}>
          <DialogPrimitive.Title className="dialog-title">{title}</DialogPrimitive.Title>
          {description ? (
            <DialogPrimitive.Description className="dialog-description">{description}</DialogPrimitive.Description>
          ) : null}
          {children}
          {footer ? <div className="dialog-footer">{footer}</div> : null}
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
```

Create `web/src/console/feedback/ConfirmDialog.tsx`:

```tsx
import { Dialog } from "./Dialog";

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  message,
  confirmLabel,
  pending = false,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  message: string;
  confirmLabel: string;
  pending?: boolean;
  onConfirm: () => void;
}) {
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      footer={
        <>
          <button className="secondary-button" onClick={() => onOpenChange(false)} disabled={pending}>
            Cancel
          </button>
          <button className="primary-button" onClick={onConfirm} disabled={pending}>
            {confirmLabel}
          </button>
        </>
      }
    >
      <p className="dialog-message">{message}</p>
    </Dialog>
  );
}
```

- [ ] **Step 6: Add dialog styles to styles.css**

```css
/* ---------- Dialogs ---------- */
.dialog-overlay { position: fixed; inset: 0; z-index: 70; background: rgb(0 0 0 / 55%); }
.dialog-content { position: fixed; left: 50%; top: 50%; transform: translate(-50%, -50%); z-index: 71; width: min(520px, calc(100vw - 40px)); max-height: 88vh; overflow-y: auto; border-radius: var(--radius-dialog); border: 1px solid var(--line-soft); background: var(--surface); padding: 20px; box-shadow: 0 18px 60px rgb(0 0 0 / 55%); }
.dialog-title { margin: 0; font-size: 16px; font-weight: 700; letter-spacing: -0.01em; }
.dialog-description { color: var(--ink-soft); font-size: 12.5px; line-height: 1.55; margin: 8px 0 0; }
.dialog-message { color: var(--ink-soft); font-size: 13px; line-height: 1.6; margin: 14px 0 0; }
.dialog-footer { display: flex; justify-content: flex-end; gap: 8px; margin-top: 18px; }
```

- [ ] **Step 7: Implement useCommand (toast-wired mutations)**

Create `web/src/console/useCommand.ts`:

```ts
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

  const run = useCallback(async (...args: TArgs): Promise<TResult | undefined> => {
    setPending(true);
    try {
      const result = await actionRef.current(...args);
      if (optionsRef.current?.successMessage) toast.success(optionsRef.current.successMessage);
      optionsRef.current?.onSuccess?.(result);
      return result;
    } catch (err) {
      const message = errorMessage(err);
      toast.error(message, { retry: optionsRef.current?.retry ? () => optionsRef.current!.retry!(...args) : undefined });
      optionsRef.current?.onError?.(message);
      return undefined;
    } finally {
      setPending(false);
    }
  }, [toast]);

  return { pending, run };
}
```

- [ ] **Step 8: Run tests, lint, commit**

Run: `cd web && npx vitest run src/console/feedback/Toast.test.tsx && npm run lint`
Expected: PASS, clean.

```bash
git add web/src/console/feedback web/src/console/useCommand.ts web/src/styles.css
git commit -m "feat(web): add toast, confirmation, and command primitives"
```

---

### Task 0.5: Display primitives

**Files:**
- Create: `web/src/console/ui/StatusBadge.tsx`
- Create: `web/src/console/ui/EmptyState.tsx`
- Create: `web/src/console/ui/ErrorState.tsx`
- Create: `web/src/console/ui/Skeleton.tsx`
- Create: `web/src/console/ui/FormField.tsx`
- Create: `web/src/console/ui/RelativeTime.tsx`
- Create: `web/src/console/ui/MetricCard.tsx`
- Create: `web/src/console/ui/DataTable.tsx`
- Test: `web/src/console/ui/primitives.test.tsx`

**Interfaces:**
- `StatusBadge({ tone, label }: { tone: "critical" | "high" | "medium" | "low" | "info" | "success"; label: string })`
- `EmptyState({ title, body, action }: { title: string; body?: string; action?: ReactNode })`
- `ErrorState({ message, onRetry, onBack }: { message: string; onRetry?: () => void; onBack?: () => void })`
- `Skeleton({ height, width }: { height: number; width?: number | string })`
- `FormField({ label, htmlFor, hint, error, required, children }: ...)`
- `RelativeTime({ time, locale }: { time: string; locale: Locale })` — uses `relativeTime` from `presentation.ts` with the app locale.
- `MetricCard({ label, value, detail, tone, to }: ...)`
- `DataTable<T>({ columns, rows, rowKey, emptyTitle, emptyBody, loading, error, onRetry, onRowClick })` — columns: `{ key: string; header: string; render: (row: T) => ReactNode; sortValue?: (row: T) => string | number }`, client-side sortable by clicking headers.

- [ ] **Step 1: Write failing tests**

Create `web/src/console/ui/primitives.test.tsx`:

```tsx
import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { StatusBadge } from "./StatusBadge";
import { EmptyState } from "./EmptyState";
import { ErrorState } from "./ErrorState";
import { Skeleton } from "./Skeleton";
import { FormField } from "./FormField";
import { DataTable } from "./DataTable";

describe("console UI primitives", () => {
  it("renders a tone badge with the label", () => {
    render(<StatusBadge tone="high" label="HIGH" />);
    const badge = screen.getByText("HIGH");
    expect(badge.className).toContain("tone-high");
  });

  it("renders an empty state with title and action", () => {
    render(<EmptyState title="Nothing here" action={<button>Add</button>} />);
    expect(screen.getByText("Nothing here")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Add" })).toBeTruthy();
  });

  it("renders an error state with retry", () => {
    let clicks = 0;
    render(<ErrorState message="boom" onRetry={() => { clicks += 1; }} />);
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(clicks).toBe(1);
  });

  it("renders a skeleton with aria-busy", () => {
    render(<Skeleton height={40} />);
    expect(screen.getByRole("status").getAttribute("aria-busy")).toBe("true");
  });

  it("associates form labels with inputs and shows errors", () => {
    render(
      <FormField label="Tag" htmlFor="tag" error="Required">
        <input id="tag" />
      </FormField>,
    );
    expect(screen.getByLabelText("Tag")).toBeTruthy();
    expect(screen.getByText("Required")).toBeTruthy();
  });

  it("sorts DataTable rows on header click", () => {
    type Row = { id: string; n: number };
    const rows: Row[] = [{ id: "a", n: 2 }, { id: "b", n: 1 }];
    render(
      <DataTable
        columns={[{ key: "n", header: "N", render: (r) => r.n, sortValue: (r) => r.n }]}
        rows={rows}
        rowKey={(r) => r.id}
      />,
    );
    fireEvent.click(screen.getByRole("columnheader", { name: "N" }));
    const cells = screen.getAllByRole("cell");
    expect(cells[0].textContent).toBe("1");
  });
});
```

- [ ] **Step 2: Run to verify they fail**

Run: `cd web && npx vitest run src/console/ui/primitives.test.tsx`
Expected: FAIL (modules missing).

- [ ] **Step 3: Implement the components**

Create each file:

`StatusBadge.tsx`:

```tsx
export type Tone = "critical" | "high" | "medium" | "low" | "info" | "success";

export function StatusBadge({ tone, label }: { tone: Tone; label: string }) {
  return <span className={`tone tone-${tone}`}>{label}</span>;
}
```

`EmptyState.tsx`:

```tsx
import type { ReactNode } from "react";

export function EmptyState({
  title,
  body,
  action,
}: {
  title: string;
  body?: string;
  action?: ReactNode;
}) {
  return (
    <div className="empty-state">
      <strong>{title}</strong>
      {body ? <p>{body}</p> : null}
      {action ? <div className="empty-action">{action}</div> : null}
    </div>
  );
}
```

`ErrorState.tsx`:

```tsx
export function ErrorState({
  message,
  onRetry,
  onBack,
}: {
  message: string;
  onRetry?: () => void;
  onBack?: () => void;
}) {
  return (
    <div className="error-state" role="alert">
      <strong>Something went wrong</strong>
      <p>{message}</p>
      <div className="error-actions">
        {onRetry ? <button className="secondary-button" onClick={onRetry}>Retry</button> : null}
        {onBack ? <button className="secondary-button" onClick={onBack}>Back</button> : null}
      </div>
    </div>
  );
}
```

`Skeleton.tsx`:

```tsx
export function Skeleton({ height, width = "100%" }: { height: number; width?: number | string }) {
  return (
    <div
      role="status"
      aria-busy="true"
      aria-label="Loading"
      className="skeleton"
      style={{ height, width }}
    />
  );
}
```

`FormField.tsx`:

```tsx
import type { ReactNode } from "react";

export function FormField({
  label,
  htmlFor,
  hint,
  error,
  required = false,
  children,
}: {
  label: string;
  htmlFor: string;
  hint?: string;
  error?: string;
  required?: boolean;
  children: ReactNode;
}) {
  return (
    <div className="form-field">
      <label htmlFor={htmlFor}>
        {label}
        {required ? <span aria-hidden="true"> *</span> : null}
      </label>
      {children}
      {hint && !error ? <p className="form-hint">{hint}</p> : null}
      {error ? <p className="form-error" role="alert">{error}</p> : null}
    </div>
  );
}
```

`RelativeTime.tsx`:

```tsx
import { relativeTime } from "../../presentation";
import type { Locale } from "../../i18n/messages";

export function RelativeTime({ time, locale }: { time: string; locale: Locale }) {
  return <time dateTime={time}>{relativeTime(time, locale)}</time>;
}
```

`MetricCard.tsx`:

```tsx
import { Link } from "react-router-dom";
import type { Tone } from "./StatusBadge";

export function MetricCard({
  label,
  value,
  detail,
  tone = "info",
  to,
}: {
  label: string;
  value: string;
  detail?: string;
  tone?: Tone;
  to?: string;
}) {
  const body = (
    <div className={`metric metric-${tone}`}>
      <span>{label}</span>
      <strong>{value}</strong>
      {detail ? <small>{detail}</small> : null}
    </div>
  );
  return to ? <Link to={to} className="metric-link">{body}</Link> : body;
}
```

`DataTable.tsx`:

```tsx
import { useMemo, useState, type ReactNode } from "react";
import { EmptyState } from "./EmptyState";
import { ErrorState } from "./ErrorState";
import { Skeleton } from "./Skeleton";

export interface Column<T> {
  key: string;
  header: string;
  render: (row: T) => ReactNode;
  sortValue?: (row: T) => string | number;
}

export function DataTable<T>({
  columns,
  rows,
  rowKey,
  emptyTitle,
  emptyBody,
  loading = false,
  error,
  onRetry,
  onRowClick,
}: {
  columns: Array<Column<T>>;
  rows: T[];
  rowKey: (row: T) => string;
  emptyTitle: string;
  emptyBody?: string;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  onRowClick?: (row: T) => void;
}) {
  const [sort, setSort] = useState<{ key: string; dir: 1 | -1 } | null>(null);

  const sorted = useMemo(() => {
    if (!sort) return rows;
    const column = columns.find((c) => c.key === sort.key);
    if (!column?.sortValue) return rows;
    return [...rows].sort((a, b) => {
      const av = column.sortValue!(a);
      const bv = column.sortValue!(b);
      if (av < bv) return -1 * sort.dir;
      if (av > bv) return 1 * sort.dir;
      return 0;
    });
  }, [rows, sort, columns]);

  if (error) {
    return <ErrorState message={error} onRetry={onRetry} />;
  }
  if (loading) {
    return <Skeleton height={180} />;
  }
  if (sorted.length === 0) {
    return <EmptyState title={emptyTitle} body={emptyBody} />;
  }
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            {columns.map((column) => (
              <th
                key={column.key}
                role="columnheader"
                aria-sort={sort?.key === column.key ? (sort.dir === 1 ? "ascending" : "descending") : undefined}
              >
                {column.sortValue ? (
                  <button
                    className="sort-button"
                    onClick={() =>
                      setSort((prev) =>
                        prev?.key === column.key
                          ? { key: column.key, dir: prev.dir === 1 ? -1 : 1 }
                          : { key: column.key, dir: 1 },
                      )
                    }
                  >
                    {column.header}
                  </button>
                ) : (
                  column.header
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {sorted.map((row) => (
            <tr key={rowKey(row)} onClick={onRowClick ? () => onRowClick(row) : undefined}>
              {columns.map((column) => (
                <td key={column.key}>{column.render(row)}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```

- [ ] **Step 4: Add supporting styles to styles.css**

```css
/* ---------- UI primitives ---------- */
.error-state { border: 1px solid var(--line); border-left: 3px solid var(--red); background: var(--surface); padding: 18px; }
.error-state strong { font-size: 13px; }
.error-state p { color: var(--ink-soft); font-size: 12px; margin: 8px 0 14px; }
.error-actions, .empty-action { display: flex; gap: 8px; }
.form-field { display: grid; gap: 6px; }
.form-field > label { color: var(--ink-muted); font-size: 9px; font-weight: 700; text-transform: uppercase; letter-spacing: .07em; }
.form-hint { color: var(--ink-muted); font-size: 10px; margin: 0; }
.form-error { color: var(--red); font-size: 11px; margin: 0; }
.metric-link { text-decoration: none; color: inherit; display: block; }
.metric-link:hover .metric { border-color: var(--brand-dim); }
.sort-button { border: 0; background: transparent; color: inherit; font: inherit; text-transform: inherit; letter-spacing: inherit; padding: 0; cursor: pointer; }
.sort-button:hover { color: var(--ink); }
tr[onclick] { cursor: pointer; }
tr[onclick]:hover td { background: var(--raised); }
```

- [ ] **Step 5: Run tests, lint, commit**

```bash
git add web/src/console/ui web/src/styles.css
git commit -m "feat(web): add console display primitives"
```

---

### Task 0.6: App shell — route config, breadcrumbs, site switcher, sidebar, /executions

**Files:**
- Create: `web/src/console/layout/PageHeader.tsx`
- Create: `web/src/console/layout/SiteSwitcher.tsx`
- Create: `web/src/console/layout/PageTrail.tsx` (trail context)
- Create: `web/src/console/state/PrincipalProvider.tsx`
- Create: `web/src/console/state/SiteContext.tsx`
- Create: `web/src/console/pages/ExecutionsPage.tsx`
- Modify: `web/src/console/usePrincipal.ts` (consume provider)
- Modify: `web/src/console/layout/ConsoleLayout.tsx` (skip link, provider mounting)
- Modify: `web/src/console/layout/Sidebar.tsx` (icons, Executions, aria-current, aria-label)
- Modify: `web/src/App.tsx` (mount providers + ToastProvider, /executions route, prune dead code)
- Modify: `web/src/i18n/messages.ts` (EN+VI keys: `nav.executions`, `site.switcherLabel`, `site.allSites`, `topbar.searchPlaceholder`, `executions.title`, `executions.lead`, `executions.empty`, `pageTitle.executions`, `nav.mainNavigation`)
- Test: `web/src/console/layout/SiteSwitcher.test.tsx`, `web/src/console/layout/PageHeader.test.tsx`

**Interfaces:**
- Produces:
  - `usePageTrail(): TrailItem[]` + `PageTrailProvider` (each page provides its trail; `PageHeader` renders it).
  - `PageHeader({ title, actions, principal })` — breadcrumbs + h1 + actions + right cluster (site switcher, search, operator chip).
  - `SiteProvider({ principal, children })` + `useSite(): { siteId: string | undefined; setSiteId: (id: string) => void }`.
  - `PrincipalProvider({ children })` + `usePrincipal(): { data?: Principal }` (memoized, single /me).

- [ ] **Step 1: Write failing tests**

Create `web/src/console/layout/SiteSwitcher.test.tsx`:

```tsx
import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SiteSwitcher } from "./SiteSwitcher";
import { I18nProvider } from "../../i18n/I18nProvider";

describe("SiteSwitcher", () => {
  it("renders nothing for a single-site principal", () => {
    const { container } = render(
      <I18nProvider>
        <SiteSwitcher siteIds={["s1"]} value="s1" onChange={() => {}} />
      </I18nProvider>,
    );
    expect(container.firstChild).toBeNull();
  });

  it("switches site for multi-site principals", () => {
    let picked = "s1";
    render(
      <I18nProvider>
        <SiteSwitcher siteIds={["s1", "s2"]} value={picked} onChange={(id) => { picked = id; }} />
      </I18nProvider>,
    );
    fireEvent.change(screen.getByLabelText(/site/i), { target: { value: "s2" } });
    expect(picked).toBe("s2");
  });
});
```

Create `web/src/console/layout/PageHeader.test.tsx`:

```tsx
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { PageHeader } from "./PageHeader";
import { I18nProvider } from "../../i18n/I18nProvider";
import { SiteProvider } from "../state/SiteContext";

describe("PageHeader", () => {
  it("renders title, breadcrumbs, and actions", () => {
    render(
      <I18nProvider>
        <SiteProvider>
          <MemoryRouter>
            <PageHeader
              title="Incident IN-1042"
              trail={[{ label: "Incidents", to: "/incidents" }, { label: "IN-1042" }]}
              actions={<button>Resolve</button>}
            />
          </MemoryRouter>
        </SiteProvider>
      </I18nProvider>,
    );
    expect(screen.getByRole("heading", { level: 1, name: "Incident IN-1042" })).toBeTruthy();
    expect(screen.getByText("Incidents")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Resolve" })).toBeTruthy();
  });
});
```

- [ ] **Step 2: Run to verify they fail**

Run: `cd web && npx vitest run src/console/layout/SiteSwitcher.test.tsx src/console/layout/PageHeader.test.tsx`
Expected: FAIL (missing modules).

- [ ] **Step 3: Implement state providers**

Create `web/src/console/state/SiteContext.tsx`:

```tsx
import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import type { Principal } from "../../types";

interface SiteValue {
  siteId: string | undefined;
  setSiteId: (id: string) => void;
}

const SiteContext = createContext<SiteValue | null>(null);

export function SiteProvider({ principal, children }: { principal?: Principal; children: ReactNode }) {
  const [siteId, setSiteId] = useState<string | undefined>(undefined);

  useEffect(() => {
    if (!siteId && principal?.site_ids[0]) {
      setSiteId(principal.site_ids[0]);
    }
  }, [principal, siteId]);

  return (
    <SiteContext.Provider value={{ siteId, setSiteId }}>{children}</SiteContext.Provider>
  );
}

export function useSite(): SiteValue {
  const value = useContext(SiteContext);
  if (!value) throw new Error("useSite must be used within SiteProvider");
  return value;
}
```

Create `web/src/console/state/PrincipalProvider.tsx`:

```tsx
import { createContext, useContext, type ReactNode } from "react";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import type { Principal } from "../../types";

const PrincipalContext = createContext<Principal | undefined>(undefined);

export function PrincipalProvider({ children }: { children: ReactNode }) {
  const { data } = useQuery(() => api.principal());
  return (
    <PrincipalContext.Provider value={data}>{children}</PrincipalContext.Provider>
  );
}

export function usePrincipalValue(): Principal | undefined {
  return useContext(PrincipalContext);
}
```

Rewrite `web/src/console/usePrincipal.ts` to consume the provider (keep `{ data }` shape):

```ts
import { usePrincipalValue } from "./state/PrincipalProvider";

export function usePrincipal(): { data: Principal | undefined } {
  return { data: usePrincipalValue() };
}
```

(`Principal` import comes from `../../types` in the provider file.)

- [ ] **Step 4: Implement trail context and PageHeader**

Create `web/src/console/layout/PageTrail.tsx`:

```tsx
import { createContext, useContext, type ReactNode } from "react";
import type { TrailItem } from "./Breadcrumbs";

const TrailContext = createContext<TrailItem[]>([]);

export function PageTrailProvider({ trail, children }: { trail: TrailItem[]; children: ReactNode }) {
  return <TrailContext.Provider value={trail}>{children}</TrailContext.Provider>;
}

export function usePageTrail(): TrailItem[] {
  return useContext(TrailContext);
}
```

Create `web/src/console/layout/PageHeader.tsx`:

```tsx
import type { ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { useI18n } from "../../i18n/I18nProvider";
import type { Principal } from "../../types";
import { Breadcrumbs } from "./Breadcrumbs";
import { SiteSwitcher } from "./SiteSwitcher";
import { usePageTrail } from "./PageTrail";
import { useSite } from "../state/SiteContext";

export function PageHeader({
  title,
  actions,
  principal,
}: {
  title: string;
  actions?: ReactNode;
  principal?: Principal;
}) {
  const { t } = useI18n();
  const navigate = useNavigate();
  const trail = usePageTrail();
  const { siteId, setSiteId } = useSite();

  return (
    <header className="page-header">
      {trail.length > 0 ? <Breadcrumbs trail={trail} /> : null}
      <div className="page-header-row">
        <h1>{title}</h1>
        <div className="page-header-right">
          {actions}
          <SiteSwitcher
            siteIds={principal?.site_ids ?? []}
            value={siteId}
            onChange={setSiteId}
          />
          <form
            role="search"
            onSubmit={(event) => {
              event.preventDefault();
              const input = event.currentTarget.elements.namedItem("q") as HTMLInputElement;
              const query = input.value.trim();
              if (query) navigate(`/search?q=${encodeURIComponent(query)}`);
            }}
          >
            <input
              name="q"
              aria-label={t("topbar.searchPlaceholder")}
              placeholder={t("topbar.searchPlaceholder")}
              className="global-search"
            />
          </form>
          <div className="operator">
            <span className="presence" />
            <span>
              <strong>{principal?.display_name ?? t("topbar.connecting")}</strong>
              <small>
                {principal?.site_ids.length
                  ? t("topbar.siteScope", { count: principal.site_ids.length })
                  : t("topbar.allSiteScope")}
              </small>
            </span>
          </div>
        </div>
      </div>
    </header>
  );
}
```

Create `web/src/console/layout/SiteSwitcher.tsx`:

```tsx
import { useI18n } from "../../i18n/I18nProvider";

export function SiteSwitcher({
  siteIds,
  value,
  onChange,
}: {
  siteIds: string[];
  value?: string;
  onChange: (siteId: string) => void;
}) {
  const { t } = useI18n();
  if (siteIds.length < 2) return null;
  return (
    <label className="site-switcher">
      <span className="eyebrow">{t("site.switcherLabel")}</span>
      <select value={value ?? ""} onChange={(e) => onChange(e.target.value)}>
        {siteIds.map((id) => (
          <option key={id} value={id}>{id}</option>
        ))}
      </select>
    </label>
  );
}
```

- [ ] **Step 5: Add i18n keys**

In `messages.ts`, add to the `en` object (near the `nav.*` keys):

```ts
  "nav.executions": "Executions",
  "nav.mainNavigation": "Main navigation",
  "site.switcherLabel": "Site",
  "site.allSites": "All sites",
  "topbar.searchPlaceholder": "Search assets, incidents, documents…",
  "pageTitle.executions": "Executions",
  "executions.title": "Executions",
  "executions.lead": "Every procedure execution across your sites, with live state and progress.",
  "executions.empty": "No executions yet.",
  "executions.state": "State",
  "executions.incident": "Incident",
  "executions.asset": "Asset",
  "executions.progress": "Progress",
  "executions.started": "Started",
```

and the VI equivalents:

```ts
  "nav.executions": "Quy trình thực hiện",
  "nav.mainNavigation": "Điều hướng chính",
  "site.switcherLabel": "Địa điểm",
  "site.allSites": "Tất cả địa điểm",
  "topbar.searchPlaceholder": "Tìm tài sản, sự cố, tài liệu…",
  "pageTitle.executions": "Quy trình thực hiện",
  "executions.title": "Quy trình thực hiện",
  "executions.lead": "Mọi quy trình thực hiện trên các địa điểm của bạn, kèm trạng thái và tiến độ trực tiếp.",
  "executions.empty": "Chưa có quy trình thực hiện nào.",
  "executions.state": "Trạng thái",
  "executions.incident": "Sự cố",
  "executions.asset": "Tài sản",
  "executions.progress": "Tiến độ",
  "executions.started": "Bắt đầu",
```

- [ ] **Step 6: Create ExecutionsPage**

Create `web/src/console/pages/ExecutionsPage.tsx`:

```tsx
import { Link } from "react-router-dom";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { DataTable } from "../ui/DataTable";
import { StatusBadge, type Tone } from "../ui/StatusBadge";
import type { Execution } from "../../types";

const EXECUTION_TONE: Record<string, Tone> = {
  ASSIGNED: "info",
  IN_PROGRESS: "medium",
  COMPLETED: "success",
};

export function ExecutionsPage() {
  const { t, locale } = useI18n();
  const executions = useQuery(() => api.listExecutions().then((list) => list.items));

  return (
    <PageTrailProvider trail={[]}>
      <section>
        <PageHeader title={t("pageTitle.executions")} />
        <p className="section-lead">{t("executions.lead")}</p>
        <DataTable<Execution>
          columns={[
            {
              key: "purpose",
              header: t("executions.title"),
              render: (execution) => (
                <Link to={`/executions/${execution.id}`} className="strong">
                  {execution.purpose}
                </Link>
              ),
              sortValue: (e) => e.purpose,
            },
            {
              key: "incident",
              header: t("executions.incident"),
              render: (execution) =>
                execution.incident_id ? (
                  <Link to={`/incidents/${execution.incident_id}`}>{execution.asset_tag ?? execution.incident_id}</Link>
                ) : (
                  <span className="muted">—</span>
                ),
            },
            {
              key: "asset",
              header: t("executions.asset"),
              render: (execution) => execution.asset_tag ?? "—",
            },
            {
              key: "state",
              header: t("executions.state"),
              render: (execution) => (
                <StatusBadge tone={EXECUTION_TONE[execution.state] ?? "info"} label={execution.state.replace("_", " ")} />
              ),
              sortValue: (e) => e.state,
            },
            {
              key: "progress",
              header: t("executions.progress"),
              render: (execution) => {
                const total = execution.steps?.length ?? 0;
                const done = execution.steps?.filter((s) => s.state === "COMPLETED").length ?? 0;
                const pct = total ? Math.round((done / total) * 100) : 0;
                return `${pct}%`;
              },
              sortValue: (e) => e.steps?.filter((s) => s.state === "COMPLETED").length ?? 0,
            },
          ]}
          rows={executions.data ?? []}
          rowKey={(execution) => execution.id}
          emptyTitle={t("executions.empty")}
          loading={executions.loading}
          error={executions.error}
          onRetry={() => void executions.refetch()}
        />
      </section>
    </PageTrailProvider>
  );
}
```

Note: `Execution` (types.ts:63) has no `site_id` or `created_at`, so the list is org-wide and has no date column. When the backend adds site scope or timestamps, extend the `DataTable` columns and pass `[siteId]` as the `useQuery` dep. `locale` is currently unused here but is part of the standard page shape; omit it if lint flags unused locals (remove from the destructure in that case).

- [ ] **Step 7: Update ConsoleLayout, Sidebar, App**

`ConsoleLayout.tsx` becomes:

```tsx
import { Outlet } from "react-router-dom";
import { Sidebar } from "./Sidebar";
import { usePrincipal } from "../usePrincipal";
import { SiteProvider } from "../state/SiteContext";

export function ConsoleLayout() {
  const { data: principal } = usePrincipal();
  return (
    <SiteProvider principal={principal}>
      <a href="#main-content" className="skip-link">Skip to content</a>
      <div className="app-shell">
        <Sidebar principal={principal} />
        <main id="main-content" style={{ minWidth: 0, padding: "0 28px 40px" }}>
          <Outlet />
        </main>
      </div>
    </SiteProvider>
  );
}
```

`Sidebar.tsx` changes:
- Replace `const SECTIONS` with icons (Phosphor: `House`, `WarningCircle`, `ListChecks`, `ArrowLeftRight`, `Books`, `MagnifyingGlass`, `FileText`, `Play`, `GitBranch`, `Gauge`) and add `{ to: "/executions", key: "nav.executions" }` under Operations after Incidents.
- `aria-label={t("nav.overview")}` → `aria-label={t("nav.mainNavigation")}`.
- Add `aria-current={location.pathname === item.to ? "page" : undefined}` on each Link.
- Keep lang switch, safety boundary, logout (dedupe: Sidebar keeps its `signOut`, App's duplicate is deleted in this task).

`App.tsx` changes:
- Remove the unused imports (`api`, `relativeTime`, `severityTone`, `FormEvent`, `useCallback`, `useEffect`, `useMemo`, `useState`, `PlaceholderPage`, `EmptyRow`, `EvidenceLinks`, `useI18n`), the duplicate `signOut`, and the dead `View`/`viewTitle`.
- Wrap routes: `<PrincipalProvider><ToastProvider><BrowserRouter>...` — `PrincipalProvider` must be inside `BrowserRouter`? It doesn't use router hooks (uses `useQuery` only) — mount it outside the router:

```tsx
export function App() {
  return (
    <PrincipalProvider>
      <ToastProvider>
        <BrowserRouter>
          <Routes>
            <Route element={<ConsoleLayout />}>
              ...
              <Route path="/executions" element={<ExecutionsPage />} />
              ...
```

- Add skip-link CSS: `.skip-link { position: absolute; left: -9999px; top: 0; z-index: 100; background: var(--surface); color: var(--ink); padding: 10px 16px; } .skip-link:focus { left: 16px; top: 16px; }` and `.page-header` / `.page-header-row` / `.page-header-right` / `.global-search` / `.site-switcher` styles to `styles.css`.

- [ ] **Step 8: Run tests, lint, commit**

Run: `cd web && npx vitest run src/console && npm run lint && npm test`
Expected: all green. Existing page tests that render `<Topbar>` still pass (Topbar stays for phases 2-3 pages; Dashboard migrates in Task 1.1).

```bash
git add web/src/console/layout web/src/console/state web/src/console/usePrincipal.ts web/src/console/pages/ExecutionsPage.tsx web/src/App.tsx web/src/styles.css web/src/i18n/messages.ts
git commit -m "feat(web): console shell with site scope, page headers, and executions list"
```

---

### Task 1.1: Dashboard redesign

**Files:**
- Modify: `web/src/console/pages/DashboardPage.tsx` (rewrite)
- Modify: `web/src/console/components/Overview.tsx` (rewrite)
- Modify: `web/src/console/components/Metric.tsx` (replaced by MetricCard — delete usage)
- Modify: `web/src/i18n/messages.ts` (keys: `dashboard.openIncidents`, `dashboard.pendingReports`, `dashboard.assignedExecutions`, `dashboard.pendingHandovers` keep; add `dashboard.executionsInProgress`, `dashboard.myQueue`, `dashboard.myQueueEmpty`, `dashboard.viewAll`, `dashboard.activeIncidents`, `dashboard.table.incident`, `dashboard.table.asset`, `dashboard.table.severity`, `dashboard.table.state`, `dashboard.table.age` + VI)
- Test: `web/src/console/pages/DashboardPage.test.tsx` (extend)

**Behavior:**
- KPIs (real): Open incidents, Executions in progress, Critical assets, Pending handovers — as `MetricCard`s with links (`/incidents`, `/executions`, `/assets`, `/handovers`).
- **My queue** panel: incidents with `state === "IN_PROGRESS"` (fallback for assignment), clickable rows → `/incidents/:id`; empty → `EmptyState`.
- **Active incidents** panel: `DataTable` of open incidents (excludes RESOLVED), columns: number (link), summary, asset, severity (StatusBadge), state (StatusBadge), age (RelativeTime).
- Shortcuts: keep only permission-relevant cards, give real counts as details, correct routes; remove "Tap to open".
- Errors: `ErrorState` per failed query with Retry (not a toast + zeroed dashboard); loading = skeletons.
- Tests: assert no "Unsafe" fake metric; assert RESOLVED rows excluded; assert KPI links point to real routes; assert error state renders with retry.

- [ ] **Step 1: Write/extend failing tests**

In `web/src/console/pages/DashboardPage.test.tsx`, add (the file's existing `render` helper must be updated to wrap in `I18nProvider` + `SiteProvider` + `PrincipalProvider` + `MemoryRouter`, since the redesigned page consumes `useSite` and the principal provider):

```tsx
  it("does not render fabricated metrics", () => {
    render(<DashboardPage />);
    expect(screen.queryByText(/unsafe ai actions/i)).toBeNull();
  });

  it("excludes resolved incidents from the active table", async () => {
    render(<DashboardPage />);
    await screen.findByText(/IN-1042/i);
    expect(screen.queryByText(/RESOLVED/i)).toBeNull();
  });
```

(Mock `api.incidents`/`api.assets` in this file's existing pattern; keep the original happy-path assertions passing.)

- [ ] **Step 2: Run to verify they fail**

Run: `cd web && npx vitest run src/console/pages/DashboardPage.test.tsx`
Expected: FAIL on the new assertions.

- [ ] **Step 3: Rewrite Overview**

`Overview.tsx` — props `{ incidents, openCount, executionCount, criticalAssetCount, pendingHandoverCount }`; renders the KPI `MetricCard` row + "My queue" + active-incidents `DataTable` (open only). No fake metrics. Rows navigate via `useNavigate`.

- [ ] **Step 4: Rewrite DashboardPage**

`DashboardPage.tsx` — uses `useQuery` for incidents/assets (deps `[siteId]` from `useSite`), `PageHeader`, `Overview`, `ErrorState`/`Skeleton` states, shortcut cards with real counts.

- [ ] **Step 5: Add i18n keys (EN + VI)** for the new labels listed in Files.

- [ ] **Step 6: Run tests, lint, commit**

```bash
git add web/src/console/pages/DashboardPage.tsx web/src/console/components/Overview.tsx web/src/console/pages/DashboardPage.test.tsx web/src/i18n/messages.ts
git commit -m "feat(web): redesign dashboard with real KPIs and my queue"
```

---

### Task 1.2: Incidents queue redesign

**Files:**
- Modify: `web/src/console/pages/IncidentsPage.tsx` (rewrite)
- Modify: `web/src/console/components/IncidentQueue.tsx` (rewrite as filterable table)
- Modify: `web/src/console/components/CreateIncidentForm.tsx` (rewrite with FormField/validation/i18n severity, no demo defaults)
- Modify: `web/src/i18n/messages.ts` (keys: `incidents.filter.state`, `incidents.filter.severity`, `incidents.filter.search`, `incidents.tabs.open`, `incidents.tabs.inProgress`, `incidents.tabs.resolved`, `incidents.tabs.all`, `incidents.create.title`, `incidents.create.summary`, `incidents.create.severity`, `incidents.create.success`, `incidents.create.error`, `incidents.stateLabel.*` tone map + VI)
- Test: `web/src/console/pages/IncidentsPage.test.tsx` (new)

**Behavior:**
- Toolbar: search input (client-side filter on number/summary/asset), state tabs with counts (Open / In progress / Resolved / All), severity filter select.
- `DataTable` rows: number (link), summary, asset, severity badge, state badge (localized, tone-mapped), age. Sortable by number/state/detected.
- New incident: `Dialog` + `FormField`s, required validation, i18n severity options, no defaults; success → toast + navigate to detail; failure → inline `FormField` error + toast.
- Unauthorized create: "New incident" button hidden unless `incident:create`.
- Tests: filter tabs reduce rows; unauthorized hides create; create validation blocks empty submit; failure shows error.

- [ ] **Step 1: Write failing tests** (filtering, unauthorized create hidden, empty-submit validation)

- [ ] **Step 2: Run to verify they fail**

- [ ] **Step 3: Rewrite the three files per the Behavior spec**

- [ ] **Step 4: Add i18n keys (EN + VI)**

- [ ] **Step 5: Run tests, lint, commit**

```bash
git add web/src/console/pages/IncidentsPage.tsx web/src/console/components/IncidentQueue.tsx web/src/console/components/CreateIncidentForm.tsx web/src/console/pages/IncidentsPage.test.tsx web/src/i18n/messages.ts
git commit -m "feat(web): redesign incident queue with filters and real create dialog"
```

---

### Task 1.3: Incident detail redesign

**Files:**
- Modify: `web/src/console/pages/IncidentDetailPage.tsx` (rewrite)
- Modify: `web/src/i18n/messages.ts` (keys: `incident.resolve.title`, `incident.resolve.message`, `incident.resolve.confirm`, `incident.resolve.reason`, `incident.resolve.success`, `incident.reopen`, `incident.timeline`, `incident.facts`, `incident.executions`, `incident.createExecutionSuccess`, `incident.versionConflict`, `incident.detail.goToWorkbench` + VI)
- Test: `web/src/console/pages/IncidentDetailPage.test.tsx` (extend)

**Behavior:**
- `PageHeader` with trail `[Incidents → number]`, title = summary (h1), subtitle line = number + asset.
- Facts panel: asset (link), severity badge, state badge, detected (`RelativeTime`).
- Timeline panel (new `Timeline` component, API-dependent — render state transitions locally from available data; empty → `EmptyState`).
- Executions panel: `DataTable` (purpose, state, started); "Create execution" → navigates to `/executions/:id` on success + toast; rows link to workbench.
- Actions: Resolve → `ConfirmDialog` with optional reason + success toast + version guard; Reopen when RESOLVED (permission-gated); disabled-with-reason tooltips for missing permissions.
- Remove the bare early-return error: `ErrorState` with Back/Retry keeps navigation chrome.
- Fix raw state text and `toLowerCase()` severity (use `StatusBadge`/localized labels).
- Tests: resolve opens confirm and calls api; unauthorized hides resolve; error state renders with chrome.

- [ ] **Step 1: Write/extend failing tests**

- [ ] **Step 2: Run to verify they fail**

- [ ] **Step 3: Rewrite IncidentDetailPage per the Behavior spec**

- [ ] **Step 4: Add i18n keys (EN + VI)**

- [ ] **Step 5: Run tests, lint, commit**

```bash
git add web/src/console/pages/IncidentDetailPage.tsx web/src/console/pages/IncidentDetailPage.test.tsx web/src/i18n/messages.ts
git commit -m "feat(web): redesign incident detail with confirmations and timeline"
```

---

### Task 1.4: Executions list (polish after Task 0.6)

- If Task 0.6 shipped a minimal `ExecutionsPage`, this task upgrades it to production quality: real site scope (filter by `site_id` when available), state filter tabs, progress bars, and an e2e-friendly row structure. Otherwise fold into Task 0.6.

---

### Task 1.5: Workbench redesign

**Files:**
- Modify: `web/src/console/pages/ExecutionWorkbenchPage.tsx` (rewrite)
- Modify: `web/src/console/components/StepRail.tsx` (rewrite)
- Modify: `web/src/console/components/MeasurementForm.tsx` (rewrite)
- Modify: `web/src/i18n/messages.ts` (keys: `workbench.complete`, `workbench.completeConfirm`, `workbench.confirmStep`, `workbench.confirmStepBody`, `workbench.undoStep`, `workbench.evidence`, `workbench.actions`, `workbench.observations`, `workbench.observation`, `workbench.addObservation`, `workbench.conflict.title`, `workbench.conflict.body`, `workbench.conflict.reload`, `workbench.blocked`, `workbench.noEvidence`, `workbench.step.requires`, `workbench.invalidMeasurement` + VI)
- Test: `web/src/console/pages/ExecutionWorkbenchPage.test.tsx` (extend)

**Behavior:**
- `PageHeader`: trail `[Incidents → IN-1042 → Execution]`, h1 = purpose once, actions row: Start (ASSIGNED), **Complete** (IN_PROGRESS, with `ConfirmDialog`), Draft report (only when `canWrite` and state permits), progress `ProgressBar`.
- Step rail: rows are buttons with visible text (not `disabled` on blocked — reason text beside the row so keyboard users can read it); safety-significant steps require `ConfirmDialog`; completed steps show undo for non-gated steps; prerequisite status; risk tone mapped via `riskTone(risk_level)`.
- Measurements: `FormField`-based, required + numeric validation, unit as suffix text, reset after submit, `RelativeTime` timestamps, `aria-live` "Recorded" list.
- Observations: structured rendering (fields from the observation type; no `JSON.stringify`), add-observation input (fallback: read-only list when API lacks a write endpoint).
- Evidence/actions panel: render `Execution.actions` + recorded evidence via `EvidencePanel` with `EmptyState`.
- Version conflict: on `ApiError.status === 409`, show `ConfirmDialog` "Changed by another operator" → refetch + re-apply.
- Every mutation via `useCommand` → success/error toast.
- Tests: complete action appears in IN_PROGRESS; read-only users get no dead clicks (step buttons disabled with title); safety step requires confirmation; JSON dump never appears.

- [ ] **Step 1: Write/extend failing tests**

- [ ] **Step 2: Run to verify they fail**

- [ ] **Step 3: Rewrite the three files per the Behavior spec**

- [ ] **Step 4: Add i18n keys (EN + VI)**

- [ ] **Step 5: Run tests, lint, commit**

```bash
git add web/src/console/pages/ExecutionWorkbenchPage.tsx web/src/console/components/StepRail.tsx web/src/console/components/MeasurementForm.tsx web/src/console/pages/ExecutionWorkbenchPage.test.tsx web/src/i18n/messages.ts
git commit -m "feat(web): redesign execution workbench with completion and confirmations"
```

---

## Final acceptance gate (after Task 1.5)

- [ ] `cd web && npm run lint` — clean
- [ ] `cd web && npm test` — all suites green (existing console tests migrated to new primitives stay green)
- [ ] `cd web && npm run build` — succeeds
- [ ] Manual: multi-site principal sees the SiteSwitcher; switching refetches pages
- [ ] Manual: resolve/LOTO-step/complete all require confirmation; every mutation toasts
- [ ] No raw enum strings, no `JSON.stringify`, no "Tap to open", no "00%" on any migrated page
- [ ] Sidebar shows Executions; `/executions` loads and links to workbenches
- [ ] Skip link works; one h1 per page; focus-visible rings visible

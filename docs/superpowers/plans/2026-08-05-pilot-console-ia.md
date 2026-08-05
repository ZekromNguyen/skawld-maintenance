# Pilot Console IA Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert the flat 8-view web console into a routed, role-aware pilot workbench: dashboard, asset/incident detail, execution workbench, reports library, document detail, and hybrid search.

**Architecture:** React Router v7 (`BrowserRouter`) with one console layout shell (sidebar + breadcrumbs + topbar) and per-page data fetching through a small `useApi` hook. The 1,717-line `App.tsx` is split into `web/src/console/pages/` and `web/src/console/components/`; the dark instrument theme and `styles.css` are kept, with new token-driven component classes added. The existing `request` helper already redirects to `/auth/login` with `return_to` on 401, which preserves deep links across login.

**Tech Stack:** Vite 8, React 19, TypeScript, Tailwind v4 (CSS-first), react-router-dom v7, @radix-ui/react-dialog + @radix-ui/react-tabs, @phosphor-icons/react, vitest + @testing-library/react.

## Global Constraints

- Console theme: existing `styles.css` tokens (`--bg`, `--surface`, `--brand` emerald, etc.). Do not change the visual identity.
- One accent color (emerald `#68d391`), one radius system, one theme per session (dark default; console is dark-first).
- Every visible string gets an en + vi key in `web/src/i18n/messages.ts` (both `en` and `vi` objects). No hardcoded user-facing text.
- No em-dashes (`—` / `–`) in any visible copy. Use hyphens, commas, periods.
- Role gating is by permission (`principal.permissions: string[]`), never by role name string.
- All new pages ship loading (skeleton), empty (CTA), and error (specific, non-apologetic) states.
- `tsc -b`, `vitest run`, `vite build` must stay green after every task.
- Icons: `@phosphor-icons/react` only. No hand-rolled SVG paths.
- API client pattern: `request<T>(path, init)` and `command<T>(path, value)` from `web/src/api.ts`; 401 handling is already centralized there — do not duplicate it.
- Type-check the console only: `cd web && npm run lint`.

---

### Task 1: Routing scaffold — install deps, BrowserRouter, console layout shell

**Files:**
- Modify: `web/package.json`
- Create: `web/src/console/layout/ConsoleLayout.tsx`
- Create: `web/src/console/layout/Sidebar.tsx`
- Create: `web/src/console/layout/Topbar.tsx`
- Create: `web/src/console/layout/Breadcrumbs.tsx`
- Create: `web/src/console/pages/PlaceholderPage.tsx`
- Modify: `web/src/main.tsx`
- Test: `web/src/console/layout/ConsoleLayout.test.tsx`

**Interfaces:**
- Consumes: nothing (first task).
- Produces:
  - `<ConsoleLayout/>` — renders sidebar + topbar + `<Outlet/>` (react-router layout route).
  - `<Sidebar/>` — props `{ principal: Principal | undefined; view: string }`, nav items per Global Constraints.
  - `<Topbar/>` — props `{ title: string; principal: Principal | undefined; onLocale: (l: Locale) => void; locale: Locale }`.
  - `<Breadcrumbs/>` — props `{ trail: Array<{ label: string; to?: string }> }`.
  - `<PlaceholderPage titleKey="..."/>` — temporary page used by later tasks.

- [ ] **Step 1: Install dependencies**

Run: `cd web && npm install react-router-dom@^7 @radix-ui/react-dialog @radix-ui/react-tabs`

- [ ] **Step 2: Add placeholder styles to `styles.css`**

Append to `web/src/styles.css`:

```css
/* ---------- Router shell ---------- */
.breadcrumbs { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; font-size: 11px; color: var(--ink-muted); margin-bottom: 2px; }
.breadcrumbs a { color: var(--ink-soft); text-decoration: none; }
.breadcrumbs a:hover { color: var(--ink); text-decoration: underline; }
.breadcrumbs .sep { color: var(--ink-muted); }
.nav-section { margin-top: 18px; padding: 0 6px; color: var(--ink-muted); font-size: 9px; font-weight: 700; text-transform: uppercase; letter-spacing: .12em; }
.skeleton { background: linear-gradient(90deg, var(--raised) 25%, var(--line) 50%, var(--raised) 75%); background-size: 200% 100%; animation: shimmer 1.2s infinite; border-radius: 4px; }
@keyframes shimmer { to { background-position: -200% 0; } }
.toast-error { position: fixed; bottom: 20px; right: 20px; z-index: 60; max-width: 360px; border: 1px solid var(--red); background: var(--surface); color: var(--ink); padding: 12px 16px; font-size: 13px; border-radius: 10px; box-shadow: 0 12px 40px rgba(0,0,0,.4); }
```

- [ ] **Step 3: Write the failing shell test**

Create `web/src/console/layout/ConsoleLayout.test.tsx`:

```tsx
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { ConsoleLayout } from "./ConsoleLayout";

describe("ConsoleLayout", () => {
  it("renders sidebar brand and outlet content", () => {
    render(
      <MemoryRouter initialEntries={["/incidents"]}>
        <Routes>
          <Route element={<ConsoleLayout />}>
            <Route path="/incidents" element={<div>incident page body</div>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByText("skawld")).toBeTruthy();
    expect(screen.getByText("incident page body")).toBeTruthy();
  });
});
```

- [ ] **Step 4: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/layout/ConsoleLayout.test.tsx`
Expected: FAIL — `ConsoleLayout` module not found.

- [ ] **Step 5: Create `ConsoleLayout.tsx`**

```tsx
import { Outlet } from "react-router-dom";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";
import { Breadcrumbs, type TrailItem } from "./Breadcrumbs";

export function ConsoleLayout({ trail }: { trail?: TrailItem[] }) {
  return (
    <div className="app-shell">
      <Sidebar />
      <main style={{ minWidth: 0, padding: "0 28px 40px" }}>
        {trail && trail.length > 0 ? <Breadcrumbs trail={trail} /> : null}
        <Outlet />
      </main>
    </div>
  );
}
```

Note: `app-shell` (240px sidebar grid) already exists in `styles.css`. Keep `Topbar` usage inside each page for now (pages own their title/actions) — simpler than a layout slot; `ConsoleLayout` stays minimal and later tasks pass `trail` from each page.

- [ ] **Step 6: Create `Sidebar.tsx`** (nav sections per spec; keep language switch + safety boundary + sign out from current `App.tsx` sidebar)

```tsx
import { Link } from "react-router-dom";
import { useI18n } from "../../i18n/I18nProvider";
import type { Principal } from "../../types";

const SECTIONS: Array<{ heading: string; items: Array<{ to: string; key: string; permission?: string }> }> = [
  {
    heading: "sidebar.operations",
    items: [
      { to: "/", key: "nav.overview" },
      { to: "/incidents", key: "nav.incidents" },
      { to: "/handovers", key: "nav.handover" }
    ]
  },
  {
    heading: "sidebar.knowledge",
    items: [
      { to: "/knowledge", key: "nav.knowledge" },
      { to: "/search", key: "nav.search" }
    ]
  },
  {
    heading: "sidebar.records",
    items: [{ to: "/reports", key: "nav.reports", permission: "report:write" }]
  },
  {
    heading: "sidebar.learning",
    items: [
      { to: "/demonstrations", key: "nav.demonstrations" },
      { to: "/workflows", key: "nav.workflows" }
    ]
  },
  {
    heading: "sidebar.quality",
    items: [{ to: "/quality", key: "nav.quality" }]
  }
];

export function Sidebar({ principal }: { principal?: Principal }) {
  const { t, locale, setLocale } = useI18n();
  const has = (p?: string) => (p ? principal?.permissions.includes(p) ?? false : true);
  return (
    <aside className="sidebar">
      <div className="brand">
        <span className="brand-mark">S</span>
        <span><strong>skawld</strong><small>{t("sidebar.maintenanceOps")}</small></span>
      </div>
      <nav aria-label="Primary">
        {SECTIONS.map((section) => {
          const visible = section.items.filter((item) => has(item.permission));
          if (visible.length === 0) return null;
          return (
            <div key={section.heading}>
              <span className="nav-section">{t(section.heading)}</span>
              {visible.map((item) => (
                <Link key={item.to} to={item.to} className="nav-item">
                  {t(item.key)}
                </Link>
              ))}
            </div>
          );
        })}
      </nav>
      <label className="lang-switch">
        <span className="eyebrow">{t("lang.label")}</span>
        <select value={locale} onChange={(e) => setLocale(e.target.value as Locale)}>
          <option value="en">English</option>
          <option value="vi">Tiếng Việt</option>
        </select>
      </label>
      <div className="safety-boundary">
        <span className="eyebrow">{t("sidebar.safetyBoundary")}</span>
        <strong>{t("sidebar.advisoryOnly")}</strong>
        <p>{t("sidebar.noControl")}</p>
      </div>
      <button className="logout-button" onClick={() => signOut()}>{t("nav.signOut")}</button>
    </aside>
  );
}

// Keep the existing signOut() form-POST implementation from App.tsx (Task 7 of the
// auth session) — copy it verbatim into this file as a module-scope function.
function signOut() {
  const form = document.createElement("form");
  form.method = "POST";
  form.action = "/auth/logout";
  form.style.display = "none";
  document.body.appendChild(form);
  form.submit();
}
```

- [ ] **Step 7: Create `Topbar.tsx` and `Breadcrumbs.tsx`**

```tsx
// Topbar.tsx
import { useI18n } from "../../i18n/I18nProvider";
import type { Principal } from "../../types";

export function Topbar({ title, principal }: { title: string; principal?: Principal }) {
  const { t } = useI18n();
  return (
    <header className="topbar">
      <div>
        <span className="eyebrow">{t("topbar.maintenanceOps")}</span>
        <h1>{title}</h1>
      </div>
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
    </header>
  );
}
```

```tsx
// Breadcrumbs.tsx
import { Link } from "react-router-dom";

export type TrailItem = { label: string; to?: string };

export function Breadcrumbs({ trail }: { trail: TrailItem[] }) {
  return (
    <nav aria-label="Breadcrumb" className="breadcrumbs">
      {trail.map((item, index) => (
        <span key={index} style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
          {index > 0 && <span className="sep" aria-hidden="true">/</span>}
          {item.to ? <Link to={item.to}>{item.label}</Link> : <span>{item.label}</span>}
        </span>
      ))}
    </nav>
  );
}
```

- [ ] **Step 8: Create `PlaceholderPage.tsx`**

```tsx
import { useI18n } from "../../i18n/I18nProvider";

export function PlaceholderPage({ titleKey }: { titleKey: string }) {
  const { t } = useI18n();
  return (
    <section>
      <header className="topbar">
        <div><span className="eyebrow">{t("topbar.maintenanceOps")}</span><h1>{t(titleKey)}</h1></div>
      </header>
      <div className="empty" style={{ padding: 40 }}>{t("placeholder.underConstruction")}</div>
    </section>
  );
}
```

- [ ] **Step 9: Rewrite `main.tsx` with BrowserRouter**

```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { Landing } from "./marketing/Landing";
import { I18nProvider } from "./i18n/I18nProvider";
import { App } from "./App";
import "@fontsource-variable/geist";
import "@fontsource-variable/geist-mono";
import "./styles.css";
import "./marketing/marketing.css";

// App (the routed console) is mounted by ./App.tsx's own router; /landing stays public.
const root = createRoot(document.getElementById("root")!);
if (window.location.pathname === "/landing") {
  root.render(<StrictMode><Landing /></StrictMode>);
} else {
  root.render(<StrictMode><I18nProvider><App /></I18nProvider></StrictMode>);
}
```

Note: `App.tsx` will host the router (Task 2). `main.tsx` stays as a simple gate — the `BrowserRouter` lives inside `App.tsx` so tests can mount it with `MemoryRouter` wrappers.

- [ ] **Step 10: Add new i18n keys**

In `web/src/i18n/messages.ts`, add to BOTH `en` and `vi` objects:

```ts
// en
"sidebar.operations": "Operations",
"sidebar.knowledge": "Knowledge",
"sidebar.records": "Records",
"sidebar.learning": "Learning",
"sidebar.quality": "Quality",
"sidebar.maintenanceOps": "Maintenance intelligence",
"lang.label": "Language / Ngôn ngữ",
"nav.search": "Search",
"nav.reports": "Reports",
"placeholder.underConstruction": "This page is under construction.",
// vi
"sidebar.operations": "Vận hành",
"sidebar.knowledge": "Kiến thức",
"sidebar.records": "Hồ sơ",
"sidebar.learning": "Học hỏi",
"sidebar.quality": "Chất lượng",
"sidebar.maintenanceOps": "Trí tuệ bảo trì",
"lang.label": "Ngôn ngữ",
"nav.search": "Tìm kiếm",
"nav.reports": "Báo cáo",
"placeholder.underConstruction": "Trang này đang được xây dựng.",
```

- [ ] **Step 11: Run shell test + typecheck**

Run: `cd web && npx vitest run src/console/layout/ConsoleLayout.test.tsx && npm run lint`
Expected: PASS, then clean `tsc -b`.

- [ ] **Step 12: Commit**

```bash
git add web/package.json web/package-lock.json web/src/main.tsx web/src/console web/src/i18n/messages.ts web/src/styles.css
git commit -m "feat(web): add routed console shell with sidebar sections and breadcrumbs"
```

---

### Task 2: Router inside App.tsx with route table + auth gate

**Files:**
- Modify: `web/src/App.tsx` (keep existing views for now; add router)
- Create: `web/src/console/useApi.ts`
- Test: `web/src/console/useApi.test.ts`

**Interfaces:**
- Consumes: Task 1 (`ConsoleLayout`, `PlaceholderPage`).
- Produces:
  - `useApi<T>(fetcher: () => Promise<T>): { data?: T; loading: boolean; error?: string; refetch: () => void }`
  - Route table in `App.tsx`: `<Route element={<ConsoleLayout/>}>` wrapping all console pages; `*` → `<Navigate to="/" replace/>`.

- [ ] **Step 1: Write failing test for `useApi`**

Create `web/src/console/useApi.test.ts`:

```tsx
import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { useApi } from "./useApi";

describe("useApi", () => {
  it("loads data and exposes loading/error states", async () => {
    const fetcher = vi.fn().mockResolvedValue({ items: [1, 2] });
    const { result } = renderHook(() => useApi(fetcher));
    expect(result.current.loading).toBe(true);
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.data).toEqual({ items: [1, 2] });
    expect(result.current.error).toBeUndefined();
  });

  it("captures fetch errors as strings", async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useApi(fetcher));
    await waitFor(() => expect(result.current.error).toBe("boom"));
    expect(result.current.loading).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/useApi.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `useApi.ts`**

```ts
import { useCallback, useEffect, useState } from "react";

export function useApi<T>(fetcher: () => Promise<T>) {
  const [data, setData] = useState<T | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>(undefined);

  const load = useCallback(() => {
    let cancelled = false;
    setLoading(true);
    setError(undefined);
    fetcher()
      .then((value) => { if (!cancelled) setData(value); })
      .catch((err: unknown) => { if (!cancelled) setError(err instanceof Error ? err.message : String(err)); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [fetcher]);

  useEffect(() => load(), [load]);

  return { data, loading, error, refetch: load };
}
```

- [ ] **Step 4: Add api client detail getters + report/document/handover methods**

Modify `web/src/api.ts` — append to the `api` object (keep existing keys intact):

```ts
// inside `export const api = { ... }` add:
  asset: (id: string) => request<Asset>(`/assets/${id}`),
  incident: (id: string) => request<Incident>(`/incidents/${id}`),
  reports: () => request<ListResponse<MaintenanceReport>>("/reports"),
  report: (id: string) => request<MaintenanceReport>(`/reports/${id}`),
  submitReport: (id: string) => command<MaintenanceReport>(`/reports/${id}/submit`, {}),
  approveReport: (id: string) => command<MaintenanceReport>(`/reports/${id}/approve`, {}),
  editReport: (id: string, value: unknown) => command<MaintenanceReport>(`/reports/${id}/edit`, value),
  document: (id: string) => request<KnowledgeDocument>(`/documents/${id}`),
  approveDocumentRevision: (revisionID: string) =>
    command<unknown>(`/document-revisions/${revisionID}/approve`, {}),
  retireDocumentRevision: (revisionID: string, reason: string) =>
    command<unknown>(`/document-revisions/${revisionID}/retire`, { reason }),
  requestDocumentIngestion: (revisionID: string) =>
    command<unknown>(`/document-revisions/${revisionID}/ingestion`, {}),
  resolveIncident: (incidentID: string) =>
    command<Incident>(`/incidents/${incidentID}/resolution`, {}),
  generateRecommendation: (incidentID: string) =>
    command<Recommendation>(`/incidents/${incidentID}/recommendations`, {}),
  recommendation: (id: string) => request<Recommendation>(`/recommendations/${id}`),
  recommendationFeedback: (id: string, value: { accepted: boolean; correction?: string }) =>
    command<unknown>(`/recommendations/${id}/feedback`, value),
  handover: (id: string) => request<ShiftHandover>(`/handovers/${id}`),
  submitHandover: (id: string) => command<ShiftHandover>(`/handovers/${id}/submit`, {}),
  acceptHandover: (id: string) => command<ShiftHandover>(`/handovers/${id}/accept`, {}),
  acknowledgeHandover: (id: string) => command<ShiftHandover>(`/handovers/${id}/acknowledge`, {}),
```

- [ ] **Step 5: Wire route table in `App.tsx`**

At the top of the existing `App` component body, replace the `view` state with a router render. Keep ALL existing view components (`Overview`, `QualityPanel`, `DemonstrationPanel`, `WorkflowLearningPanel`, `KnowledgePanel`, `HandoverPanel`, `AssetsTable`, `IncidentQueue`, `ExecutionPanel`) as local functions for now; they get migrated to pages in Tasks 3-10. The `App` component becomes:

```tsx
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
// ...existing imports...

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<ConsoleLayout />}>
          <Route path="/" element={<PlaceholderPage titleKey="nav.overview" />} />
          <Route path="/assets" element={<PlaceholderPage titleKey="nav.assets" />} />
          <Route path="/assets/:assetId" element={<PlaceholderPage titleKey="nav.assets" />} />
          <Route path="/incidents" element={<PlaceholderPage titleKey="nav.incidents" />} />
          <Route path="/incidents/:incidentId" element={<PlaceholderPage titleKey="nav.incidents" />} />
          <Route path="/executions/:executionId" element={<PlaceholderPage titleKey="nav.executions" />} />
          <Route path="/reports" element={<PlaceholderPage titleKey="nav.reports" />} />
          <Route path="/reports/:reportId" element={<PlaceholderPage titleKey="nav.reports" />} />
          <Route path="/handovers" element={<PlaceholderPage titleKey="nav.handover" />} />
          <Route path="/knowledge" element={<PlaceholderPage titleKey="nav.knowledge" />} />
          <Route path="/knowledge/:documentId" element={<PlaceholderPage titleKey="nav.knowledge" />} />
          <Route path="/search" element={<PlaceholderPage titleKey="nav.search" />} />
          <Route path="/quality" element={<PlaceholderPage titleKey="nav.quality" />} />
          <Route path="/demonstrations" element={<PlaceholderPage titleKey="nav.demonstrations" />} />
          <Route path="/workflows" element={<PlaceholderPage titleKey="nav.workflows" />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
```

Move the `principal` fetch (the existing `api.principal()` effect) into a tiny `usePrincipal` hook in `useApi.ts`:

```ts
import { api } from "../api";
import type { Principal } from "../types";

export function usePrincipal() {
  return useApi<Principal>(() => api.principal());
}
```

Add i18n key `nav.executions` (en: "Executions", vi: "Thực hiện bảo trì") to `messages.ts`.

- [ ] **Step 6: Run tests + lint + build**

Run: `cd web && npx vitest run && npm run lint && npm run build`
Expected: all pass (existing tests unchanged; placeholders render).

- [ ] **Step 7: Commit**

```bash
git add web/src/App.tsx web/src/api.ts web/src/console/useApi.ts web/src/i18n/messages.ts
git commit -m "feat(web): wire react-router route table and useApi data hook"
```

---

### Task 3: Migrate existing views into pages (Overview, Quality, Demonstrations, Workflows, Knowledge, Handover)

**Files:**
- Create: `web/src/console/pages/DashboardPage.tsx` (wraps existing `Overview`)
- Create: `web/src/console/pages/QualityPage.tsx`
- Create: `web/src/console/pages/DemonstrationsPage.tsx`
- Create: `web/src/console/pages/WorkflowsPage.tsx`
- Create: `web/src/console/pages/KnowledgePage.tsx`
- Create: `web/src/console/pages/HandoverPage.tsx`
- Modify: `web/src/App.tsx`
- Test: `web/src/console/pages/DashboardPage.test.tsx`

**Interfaces:**
- Consumes: Task 2 (`useApi`, `usePrincipal`), existing view components (moved from `App.tsx`).
- Produces: page components each exporting `export function <Name>Page()` that fetch their own data and render the migrated view; `App.tsx` route elements point to these pages.

- [ ] **Step 1: Move view components out of `App.tsx`**

For each of `Overview`, `QualityPanel`, `DemonstrationPanel`, `WorkflowLearningPanel`, `KnowledgePanel`, `HandoverPanel`, `AssetsTable`, `IncidentTable`, `IncidentQueue`, `ExecutionPanel`, `RecommendationCard`, `ReportCard`, `EvidenceLinks`, `HandoverSection`, `MeasurementForm`, `CreateAssetForm`, `CreateIncidentForm`, `Metric`, `EmptyRow`:
1. Cut the function from `App.tsx` into the new page/component file it belongs to.
2. Each page file imports the types it needs from `../types` and `useApi` from `../useApi`.
3. Delete the now-unused imports from `App.tsx` (keep it compiling — `npm run lint` is the check).

Mapping (component → file):
- `Metric`, `Overview` → `pages/DashboardPage.tsx`
- `QualityPanel` → `pages/QualityPage.tsx`
- `DemonstrationPanel` → `pages/DemonstrationsPage.tsx`
- `WorkflowLearningPanel` → `pages/WorkflowsPage.tsx`
- `KnowledgePanel` → `pages/KnowledgePage.tsx`
- `HandoverPanel`, `HandoverSection` → `pages/HandoverPage.tsx`
- `AssetsTable`, `CreateAssetForm` → `pages/AssetsPage.tsx` (list + create dialog, Task 5)
- `IncidentTable`, `IncidentQueue`, `CreateIncidentForm`, `ExecutionPanel`, `RecommendationCard`, `EvidenceLinks`, `ReportCard`, `MeasurementForm` → `pages/IncidentsPage.tsx` / `pages/ExecutionWorkbenchPage.tsx` (Tasks 6-7; keep in a shared `console/components/` folder: `Metric.tsx`, `EvidenceLinks.tsx`, `ReportCard.tsx`, `StateBadge.tsx`, `MeasurementForm.tsx`, `Skeleton.tsx`, `Toast.tsx`)
- `EmptyRow` → `console/components/EmptyRow.tsx`

- [ ] **Step 2: Write failing test for `DashboardPage`**

Create `web/src/console/pages/DashboardPage.test.tsx`:

```tsx
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { DashboardPage } from "./DashboardPage";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({ id: "p1", display_name: "Tester", site_ids: [], permissions: ["incident:read"] }),
    incidents: vi.fn().mockResolvedValue({ items: [] }),
    assets: vi.fn().mockResolvedValue({ items: [] })
  }
}));

describe("DashboardPage", () => {
  it("renders the role-aware dashboard title", () => {
    render(<MemoryRouter><DashboardPage /></MemoryRouter>);
    expect(screen.getByRole("heading", { level: 1 })).toBeTruthy();
  });
});
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/pages/DashboardPage.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 4: Implement `DashboardPage.tsx`** — wrapper that fetches principal + incidents + assets and renders the migrated `Overview` (which already takes these as props):

```tsx
import { usePrincipal, useApi } from "../useApi";
import { api } from "../../api";
import { Topbar } from "../layout/Topbar";
import { Overview } from "../components/Overview"; // moved from App.tsx

export function DashboardPage() {
  const { data: principal } = usePrincipal();
  const incidents = useApi(() => api.incidents());
  const assets = useApi(() => api.assets());
  const loading = !principal || incidents.loading || assets.loading;
  return (
    <section>
      <Topbar title={/* t("nav.overview") */ "Operations overview"} principal={principal} />
      {incidents.error && <div className="toast-error" role="alert">{incidents.error}</div>}
      {loading ? (
        <div style={{ display: "grid", gap: 12 }}>
          <div className="skeleton" style={{ height: 108 }} />
          <div className="skeleton" style={{ height: 108 }} />
          <div className="skeleton" style={{ height: 108 }} />
        </div>
      ) : (
        <Overview
          incidents={incidents.data?.items ?? []}
          assets={assets.data?.items ?? []}
          onOpenIncidents={() => (window.location.href = "/incidents")}
        />
      )}
    </section>
  );
}
```

`Overview` keeps its existing signature from `App.tsx` (take `incidents`, `assets`, `onOpenIncidents` props). Do NOT change its internals in this task.

- [ ] **Step 5: Implement the remaining page wrappers** (same pattern — fetch own data via `useApi`, render migrated panel, skeleton/error states):

- `QualityPage`: fetch `api.evaluationSummary()`, render `QualityPanel`.
- `DemonstrationsPage`: fetch `api.demonstrations()`, render `DemonstrationPanel` with its existing props (`demonstrations`, `selectedDemonstration`, `onSelect`, `onStart`, `onComplete`, `onReview`).
- `WorkflowsPage`: fetch `api.workflows()`, render `WorkflowLearningPanel`.
- `KnowledgePage`: fetch `api.documents(siteID)` — the existing `documents: (siteID: string) => ...` signature; render `KnowledgePanel`.
- `HandoverPage`: fetch `api.handovers()` — add this method to `api.ts` (`handovers: () => request<ListResponse<ShiftHandover>>("/handovers")`); render `HandoverPanel` with its existing props.

The migrated panels that previously received ALL data + callbacks from `App` will be re-wired incrementally: in this task render them with the fetched data and wire callbacks to the corresponding `api.*` methods. Where a callback has no page yet (e.g., asset create), leave it rendering the panel without the create action and add a TODO-free comment referencing the later task — **do not** add dead buttons; hide them when the handler is `undefined`.

- [ ] **Step 6: Point `App.tsx` routes at the new pages**

Replace the placeholder elements for `/`, `/quality`, `/demonstrations`, `/workflows`, `/knowledge`, `/handovers` with `<DashboardPage/>`, `<QualityPage/>`, `<DemonstrationsPage/>`, `<WorkflowsPage/>`, `<KnowledgePage/>`, `<HandoverPage/>`. Remove the migrated component definitions from `App.tsx` (now dead code). Keep placeholders for `/assets*`, `/incidents*`, `/executions/:id`, `/reports*`, `/search`, `/knowledge/:documentId`.

- [ ] **Step 7: Run tests + lint + build**

Run: `cd web && npx vitest run && npm run lint && npm run build`
Expected: all green. Manually check `/` renders the overview with data (API running).

- [ ] **Step 8: Commit**

```bash
git add web/src/console web/src/App.tsx web/src/api.ts
git commit -m "refactor(web): migrate existing views into routed pages"
```

---

### Task 4: Dashboard role-aware landing

**Files:**
- Modify: `web/src/console/pages/DashboardPage.tsx`
- Create: `web/src/console/components/ShortcutCard.tsx`
- Test: `web/src/console/pages/DashboardPage.test.tsx` (extend)

**Interfaces:**
- Consumes: Task 3 (`Overview`), `usePrincipal`.
- Produces: `ShortcutCard({ icon, label, detail, to })` — a `Link` card used across pages.

- [ ] **Step 1: Write failing test for role-aware shortcut rendering**

Extend `DashboardPage.test.tsx`:

```tsx
it("shows supervisor shortcut for open incidents", async () => {
  (api.incidents as ReturnType<typeof vi.fn>).mockResolvedValue({
    items: [{ id: "i1", number: "IN-1", summary: "Pump vibration", severity: "HIGH", state: "OPEN" }]
  });
  render(<MemoryRouter><DashboardPage /></MemoryRouter>);
  expect(await screen.findByText("Open incidents")).toBeTruthy();
  expect(screen.getByText("Pump vibration")).toBeTruthy();
});
```

Add `nav.openIncidents` en/vi keys. Adjust the mock in the test file to include the extra `api.incidents` shape.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/pages/DashboardPage.test.tsx`
Expected: FAIL — "Open incidents" not found.

- [ ] **Step 3: Implement `ShortcutCard.tsx`**

```tsx
import { Link } from "react-router-dom";
import type { Icon } from "@phosphor-icons/react";

export function ShortcutCard({ icon, label, detail, to }: {
  icon: Icon; label: string; detail: string; to: string;
}) {
  return (
    <Link to={to} className="shortcut-card">
      <span className="shortcut-icon"><icon weight="duotone" size={20} /></span>
      <span>
        <strong>{label}</strong>
        <small>{detail}</small>
      </span>
    </Link>
  );
}
```

Add to `styles.css`:

```css
.shortcut-card { display: flex; align-items: center; gap: 12px; border: 1px solid var(--line); border-radius: 12px; background: var(--surface); padding: 14px 16px; text-decoration: none; color: inherit; transition: border-color .15s ease, background-color .15s ease; }
.shortcut-card:hover { border-color: var(--brand-dim); background: var(--raised); }
.shortcut-icon { width: 38px; height: 38px; display: grid; place-items: center; border-radius: 10px; border: 1px solid var(--brand-dim); background: var(--sunken); color: var(--brand); }
.shortcut-card strong { display: block; font-size: 13px; }
.shortcut-card small { color: var(--ink-muted); font-size: 11px; }
```

- [ ] **Step 4: Rework `DashboardPage`** to branch by role using `principal.permissions`:

- If `permissions.includes("execution:write")` → show technician shortcuts (assigned executions, measurements) + `Overview`.
- If `permissions.includes("report:approve")` → supervisor shortcuts (open incidents, pending reports) + `Overview`.
- If `permissions.includes("handover:accept")` → manager shortcuts (pending handovers).
- Default → `Overview`.
Use `ShortcutCard` with Phosphor icons (`WarningCircle`, `ClipboardText`, `ArrowsLeftRight`, `ChartLineUp`). Keep `Overview` as the main content area below the shortcut row for every role.

- [ ] **Step 5: Run tests + lint**

Run: `cd web && npx vitest run src/console/pages/DashboardPage.test.tsx && npm run lint`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/console web/src/styles.css web/src/i18n/messages.ts
git commit -m "feat(web): role-aware dashboard landing with shortcut cards"
```

---

### Task 5: Assets list + Asset detail

**Files:**
- Create: `web/src/console/pages/AssetsPage.tsx`
- Create: `web/src/console/pages/AssetDetailPage.tsx`
- Create: `web/src/console/components/CreateAssetDialog.tsx`
- Modify: `web/src/App.tsx` (route to pages)
- Test: `web/src/console/pages/AssetDetailPage.test.tsx`

**Interfaces:**
- Consumes: Task 2 (`api.asset(id)`, `api.assets()`), `useApi`, `usePrincipal`, Radix Dialog.
- Produces: `AssetsPage` (list + create dialog, gated by `asset:create`), `AssetDetailPage` (breadcrumb `Assets / <tag>`, criticality approval gated by `asset:criticality:approve`, incidents history link to `/incidents`).

- [ ] **Step 1: Write failing test for asset detail**

Create `web/src/console/pages/AssetDetailPage.test.tsx`:

```tsx
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { AssetDetailPage } from "./AssetDetailPage";

vi.mock("../../api", () => ({
  api: {
    asset: vi.fn().mockResolvedValue({
      id: "a1", site_id: "s1", tag: "P-302", name: "Circulation Pump",
      class: "Pump", status: "OPERATIONAL", source_of_truth: "OWNED_BY_SKAWLD",
      criticality: { rating: "A", safety_impact: 5, production_impact: 4, rationale: "Critical to process" }
    })
  }
}));

describe("AssetDetailPage", () => {
  it("renders asset tag and criticality", async () => {
    render(
      <MemoryRouter initialEntries={["/assets/a1"]}>
        <Routes><Route path="/assets/:assetId" element={<AssetDetailPage />} /></Routes>
      </MemoryRouter>,
    );
    expect(await screen.findByText("P-302")).toBeTruthy();
    expect(screen.getByText("Circulation Pump")).toBeTruthy();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/pages/AssetDetailPage.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `AssetDetailPage.tsx`**

```tsx
import { useParams } from "react-router-dom";
import { useApi, usePrincipal } from "../useApi";
import { api } from "../../api";
import { Topbar } from "../layout/Topbar";
import { Breadcrumbs } from "../layout/Breadcrumbs";
import { useI18n } from "../../i18n/I18nProvider";

export function AssetDetailPage() {
  const { assetId } = useParams<{ assetId: string }>();
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const asset = useApi(() => api.asset(assetId!));
  const canApprove = principal?.permissions.includes("asset:criticality:approve") ?? false;

  if (asset.error) return <div className="toast-error" role="alert">{asset.error}</div>;
  if (asset.loading || !asset.data) {
    return <div className="skeleton" style={{ height: 200 }} />;
  }
  const value = asset.data;
  return (
    <section>
      <Breadcrumbs trail={[
        { label: t("nav.assets"), to: "/assets" },
        { label: value.tag }
      ]} />
      <Topbar title={value.tag} principal={principal} />
      <div style={{ display: "grid", gap: 16 }}>
        <div className="panel">
          <div className="panel-heading">
            <h2>{value.name}</h2>
            <span className="state-badge">{value.status}</span>
          </div>
          <div style={{ padding: 16, display: "grid", gridTemplateColumns: "repeat(auto-fit,minmax(160px,1fr))", gap: 12 }}>
            <div><small>Class</small><div className="strong">{value.class}</div></div>
            {value.manufacturer && <div><small>Manufacturer</small><div className="strong">{value.manufacturer}</div></div>}
            {value.model && <div><small>Model</small><div className="strong">{value.model}</div></div>}
            <div><small>Source</small><div className="strong">{value.source_of_truth}</div></div>
          </div>
        </div>
        {value.criticality && (
          <div className="panel">
            <div className="panel-heading"><h2>{t("asset.criticality")}</h2></div>
            <div style={{ padding: 16, display: "grid", gap: 10 }}>
              <div>Rating <span className={`rating-${value.criticality.rating}`}>{value.criticality.rating}</span></div>
              <div>Safety impact <span className="mono">{value.criticality.safety_impact}</span></div>
              <div>Production impact <span className="mono">{value.criticality.production_impact}</span></div>
              <p>{value.criticality.rationale}</p>
            </div>
          </div>
        )}
        {canApprove && <button className="btn-primary" onClick={() => void api.approveAssetCriticality(value.id)}>{t("asset.approveCriticality")}</button>}
      </div>
    </section>
  );
}
```

Add `api.approveAssetCriticality` if missing in `api.ts` (check existing `assets` methods first — reuse whatever exists; if absent, add `approveAssetCriticality: (assetID: string) => command<unknown>(\`/assets/${assetID}/criticality-approvals\`, {})`).

- [ ] **Step 4: Implement `AssetsPage.tsx`** — list with `AssetsTable` (moved in Task 3) + `CreateAssetDialog` (Radix `Dialog` wrapping the moved `CreateAssetForm`), create gated by `asset:create`.

- [ ] **Step 5: Point routes** in `App.tsx`: `/assets` → `AssetsPage`, `/assets/:assetId` → `AssetDetailPage`.

- [ ] **Step 6: Add i18n keys** `asset.criticality`, `asset.approveCriticality`, `asset.create`, `asset.created` (en/vi).

- [ ] **Step 7: Run tests + lint + build**

Run: `cd web && npx vitest run && npm run lint && npm run build`
Expected: green.

- [ ] **Step 8: Commit**

```bash
git add web/src/console web/src/App.tsx web/src/api.ts web/src/i18n/messages.ts
git commit -m "feat(web): asset list and detail pages with criticality approval"
```

---

### Task 6: Incidents list + Incident detail

**Files:**
- Create: `web/src/console/pages/IncidentsPage.tsx`
- Create: `web/src/console/pages/IncidentDetailPage.tsx`
- Create: `web/src/console/components/CreateIncidentDialog.tsx`
- Modify: `web/src/App.tsx`, `web/src/api.ts`
- Test: `web/src/console/pages/IncidentDetailPage.test.tsx`

**Interfaces:**
- Consumes: Task 5 components (`useApi`, dialogs pattern), `api.incident(id)`, `api.createExecution`, `api.generateRecommendation`, `api.resolveIncident`.
- Produces: `IncidentDetailPage` — renders summary, severity badge, linked asset link, executions list (links to `/executions/:id`), actions gated by permission, recommendation card with feedback.

- [ ] **Step 1: Write failing test for incident detail**

Create `web/src/console/pages/IncidentDetailPage.test.tsx` (mock `api.incident` returning a HIGH OPEN incident with an execution). Assert: summary text, severity badge, "Create execution" button present when `execution:write` permission mocked, link to `/executions/ex1` present.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/pages/IncidentDetailPage.test.tsx`
Expected: FAIL.

- [ ] **Step 3: Implement `IncidentDetailPage.tsx`**

Pattern (concise):

```tsx
export function IncidentDetailPage() {
  const { incidentId } = useParams<{ incidentId: string }>();
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const incident = useApi(() => api.incident(incidentId!));
  const [busy, setBusy] = useState(false);

  if (incident.error) return <div className="toast-error" role="alert">{incident.error}</div>;
  if (incident.loading || !incident.data) return <div className="skeleton" style={{ height: 200 }} />;
  const value = incident.data;
  const perms = principal?.permissions ?? [];
  const canCreateExecution = perms.includes("execution:write");
  const canResolve = perms.includes("incident:resolve");

  const createExecution = async () => {
    setBusy(true);
    try { await api.createExecution(value.id); incident.refetch(); }
    catch (err) { /* toast via error state */ }
    finally { setBusy(false); }
  };

  return (
    <section>
      <Breadcrumbs trail={[{ label: t("nav.incidents"), to: "/incidents" }, { label: value.number }]} />
      <Topbar title={value.number} principal={principal} />
      <div style={{ display: "grid", gap: 16 }}>
        <div className="panel">
          <div className="panel-heading">
            <h2>{value.summary}</h2>
            <span className={`severity ${value.severity.toLowerCase()}`}>{value.severity}</span>
          </div>
          <div style={{ padding: 16, display: "grid", gap: 10 }}>
            <div>State <span className="state-badge">{value.state}</span></div>
            <div>
              Asset <Link to={`/assets/${value.asset_id}`} className="strong">{value.asset_tag ?? value.asset_id}</Link>
            </div>
            <div>Detected <span className="mono">{relativeTime(value.detected_at)}</span></div>
            {canCreateExecution && <button className="btn-primary" disabled={busy} onClick={createExecution}>{t("incident.createExecution")}</button>}
            {canResolve && <button className="btn-ghost" disabled={busy} onClick={() => void api.resolveIncident(value.id).then(() => incident.refetch())}>{t("incident.resolve")}</button>}
          </div>
        </div>
        {/* executions list: from api.execution(id) per execution or a listExecutions method added to api.ts:
            listExecutions: () => request<ListResponse<Execution>>("/executions") — add it, filter client-side by incident_id */}
      </div>
    </section>
  );
}
```

`relativeTime` is already exported from `presentation.ts` — reuse it.

- [ ] **Step 4: Implement `IncidentsPage.tsx`** — queue reusing moved `IncidentQueue`; each row becomes a `Link` to `/incidents/:id`; create dialog gated by `incident:create`.

- [ ] **Step 5: Add api method** `listExecutions: () => request<ListResponse<Execution>>("/executions")` and use it in the detail page (filter by `incident_id`).

- [ ] **Step 6: Point routes** in `App.tsx`: `/incidents` → `IncidentsPage`, `/incidents/:incidentId` → `IncidentDetailPage`.

- [ ] **Step 7: i18n keys** `incident.createExecution`, `incident.resolve`, `incident.executions`, `incident.openInWorkbench` (en/vi).

- [ ] **Step 8: Run tests + lint + build; commit**

Run: `cd web && npx vitest run && npm run lint && npm run build`
Commit: `feat(web): incident queue and detail page with execution creation`

---

### Task 7: Execution workbench (anchor page)

**Files:**
- Create: `web/src/console/pages/ExecutionWorkbenchPage.tsx`
- Create: `web/src/console/components/StepRail.tsx`
- Create: `web/src/console/components/EvidencePanel.tsx`
- Modify: `web/src/App.tsx`
- Test: `web/src/console/pages/ExecutionWorkbenchPage.test.tsx`

**Interfaces:**
- Consumes: Task 6 (`api.execution(id)`, `api.startExecution`, `api.completeStep`, `api.draftReport`), moved `MeasurementForm`, `EvidenceLinks`, `ReportCard`.
- Produces: `ExecutionWorkbenchPage` — the pilot anchor page.

- [ ] **Step 1: Write failing test**

Create `web/src/console/pages/ExecutionWorkbenchPage.test.tsx` — mock `api.execution` returning an execution with 3 steps (one LOTO-gated), a measurement, and evidence. Assert: step titles render, "LOTO" gate badge renders, "Draft report" button links to `/reports` (or triggers `api.draftReport`), state badge shows current state.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/pages/ExecutionWorkbenchPage.test.tsx`
Expected: FAIL.

- [ ] **Step 3: Implement `StepRail.tsx`**

```tsx
import type { Step } from "../../types";

export function StepRail({ steps, onToggle }: { steps: Step[]; onToggle: (step: Step) => void }) {
  return (
    <div className="panel">
      <div className="panel-heading"><h2>{/* t("execution.steps") */ "Steps"}</h2></div>
      <div style={{ display: "grid" }}>
        {steps.map((step) => (
          <button
            key={step.id}
            className="step-row"
            disabled={step.state === "BLOCKED"}
            onClick={() => onToggle(step)}
            style={{ display: "flex", alignItems: "center", gap: 10, width: "100%", padding: "12px 16px", border: 0, borderBottom: "1px solid var(--line)", background: "transparent", color: "inherit", textAlign: "left" }}
          >
            <span
              style={{ width: 10, height: 10, borderRadius: "50%", background: step.state === "COMPLETED" ? "var(--brand)" : step.state === "BLOCKED" ? "var(--red)" : "var(--line-soft)", flexShrink: 0 }}
            />
            <span style={{ flex: 1 }}>
              <strong>{step.title}</strong>
              {step.required_prerequisite && <small> · {step.required_prerequisite}</small>}
            </span>
            {step.risk_level !== "INFORMATIONAL" && <span className="severity high">{step.risk_level}</span>}
            {step.state === "BLOCKED" && step.blocked_reason && <span className="state-badge">{step.blocked_reason}</span>}
          </button>
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Implement `ExecutionWorkbenchPage.tsx`**

Wire: fetch execution via `useApi(() => api.execution(executionId!))`; step toggle → `api.completeStep(execution, step)` then `refetch()`; measurement add → `MeasurementForm` (moved) → `refetch()`; "Draft report" → `api.draftReport(executionId)` then navigate to `/reports/:id` (from the returned report) or to `/reports` if no id — **check the actual `draftReport` return type in `api.ts` first** and navigate accordingly. State transitions: if `state === "ASSIGNED"` show "Start execution" button (`api.startExecution`) gated by `execution:write`.

- [ ] **Step 5: Point route** `/executions/:executionId` → `ExecutionWorkbenchPage` in `App.tsx`. Make `IncidentDetailPage` execution rows link to `/executions/:id` (replace any inline panel).

- [ ] **Step 6: i18n keys** `execution.steps`, `execution.measurements`, `execution.observations`, `execution.evidence`, `execution.start`, `execution.draftReport`, `execution.completeStep` (en/vi).

- [ ] **Step 7: Run tests + lint + build; commit**

Run: `cd web && npx vitest run && npm run lint && npm run build`
Commit: `feat(web): execution workbench with LOTO step rail and evidence`

---

### Task 8: Reports library + Report detail

**Files:**
- Create: `web/src/console/pages/ReportsPage.tsx`
- Create: `web/src/console/pages/ReportDetailPage.tsx`
- Modify: `web/src/App.tsx`
- Test: `web/src/console/pages/ReportDetailPage.test.tsx`

**Interfaces:**
- Consumes: Task 2 (`api.reports`, `api.report`, `api.submitReport`, `api.approveReport`, `api.editReport`), `ReportCard` (moved), `EvidenceLinks`.
- Produces: `ReportsPage` (library with state badges), `ReportDetailPage` (structured content + submit/approve gated by `report:write` / `report:approve`).

- [ ] **Step 1: Write failing test**

Create `web/src/console/pages/ReportDetailPage.test.tsx` — mock `api.report` returning a DRAFT report with structured_content (summary, measurements, outcome, unknowns). Assert: summary renders, "Submit" button appears when `report:write` permission mocked, "Approve" appears when `report:approve` mocked.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/pages/ReportDetailPage.test.tsx`
Expected: FAIL.

- [ ] **Step 3: Implement `ReportDetailPage.tsx`**

Render: breadcrumbs `Reports / RP-<n>`, topbar, state badge, `structured_content` sections (summary, measurements list, observations, actions, outcome, unknowns), evidence via `EvidenceLinks`, and action buttons: Submit (when DRAFT and `report:write`), Approve (when SUBMITTED and `report:approve`), Edit (DRAFT and `report:write` — opens a Radix Dialog with textareas bound to `structured_content` fields, saving via `api.editReport`).

- [ ] **Step 4: Implement `ReportsPage.tsx`** — `useApi(() => api.reports())`, list with `ReportCard` rows, each a `Link` to `/reports/:id`, state badge column.

- [ ] **Step 5: Point routes** in `App.tsx`.

- [ ] **Step 6: i18n keys** `report.submit`, `report.approve`, `report.edit`, `report.summary`, `report.measurements`, `report.observations`, `report.actions`, `report.outcome`, `report.unknowns`, `report.evidence`, `report.draft`, `report.submitted`, `report.approved` (en/vi).

- [ ] **Step 7: Run tests + lint + build; commit**

Run: `cd web && npx vitest run && npm run lint && npm run build`
Commit: `feat(web): reports library and detail with lifecycle actions`

---

### Task 9: Knowledge document detail

**Files:**
- Create: `web/src/console/pages/DocumentDetailPage.tsx`
- Modify: `web/src/App.tsx`
- Test: `web/src/console/pages/DocumentDetailPage.test.tsx`

**Interfaces:**
- Consumes: Task 2 (`api.document(id)`, `api.approveDocumentRevision`, `api.retireDocumentRevision`, `api.requestDocumentIngestion`), `usePrincipal`.
- Produces: `DocumentDetailPage`.

- [ ] **Step 1: Write failing test**

Mock `api.document` returning a KnowledgeDocument with two revisions (one APPROVED, one DRAFT). Assert: title, revision list with approval badges, Approve button appears for DRAFT revision when `knowledge:approve` permission mocked.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/pages/DocumentDetailPage.test.tsx`
Expected: FAIL.

- [ ] **Step 3: Implement `DocumentDetailPage.tsx`**

Render: breadcrumbs `Knowledge / <title>`, topbar, revision list (revision, approval status badge, ingestion state, language), per-revision actions gated by permission: Approve (`knowledge:approve`), Request ingestion (`knowledge:write`), Retire (`knowledge:write`), applicability block (site/asset/class/manufacturer/model rows).

- [ ] **Step 4: Point route** `/knowledge/:documentId` → `DocumentDetailPage`; make `KnowledgePage` rows link to it.

- [ ] **Step 5: i18n keys** `document.revisions`, `document.approve`, `document.retire`, `document.requestIngestion`, `document.applicability`, `document.ingestionState` (en/vi).

- [ ] **Step 6: Run tests + lint + build; commit**

Run: `cd web && npx vitest run && npm run lint && npm run build`
Commit: `feat(web): knowledge document detail with revision lifecycle`

---

### Task 10: Hybrid search page

**Files:**
- Create: `web/src/console/pages/SearchPage.tsx`
- Modify: `web/src/App.tsx`
- Test: `web/src/console/pages/SearchPage.test.tsx`

**Interfaces:**
- Consumes: Task 2 (`api.searchKnowledge(siteID, query)`), existing `EvidenceLinks`.
- Produces: `SearchPage` — reads `?q=` from URL (`useSearchParams`), fetches evidence results, groups by `authority`/document title, plus client-side filter tabs over already-loaded incidents/documents/workflows when those lists are available. Honest scope: the API `/search` returns knowledge `Evidence` only; "grouped by type" is implemented as tabs (Evidence / Incidents / Documents / Workflows) where the latter three filter the already-loaded lists client-side.

- [ ] **Step 1: Write failing test**

Mock `api.searchKnowledge` returning two evidence items with different `authority` values. Render `SearchPage` with `initialEntries=["/search?q=pump"]`. Assert: query is echoed in the results heading, both authorities render, empty query shows the empty-state CTA.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/pages/SearchPage.test.tsx`
Expected: FAIL.

- [ ] **Step 3: Implement `SearchPage.tsx`**

```tsx
import { useSearchParams } from "react-router-dom";
import { useApi, usePrincipal } from "../useApi";
import { api } from "../../api";
import { Topbar } from "../layout/Topbar";
import { EvidenceLinks } from "../components/EvidenceLinks";

export function SearchPage() {
  const [params] = useSearchParams();
  const query = (params.get("q") ?? "").trim();
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const siteID = principal?.site_ids[0] ?? "";
  const results = useApi(
    () => (query ? api.searchKnowledge(siteID, query) : Promise.resolve({ items: [] as import("../../types").Evidence[] })),
  );
  // render: search input (form -> navigate(`/search?q=${encodeURIComponent(q)}`)),
  // heading with query, EvidenceLinks over grouped results, empty state when query && !loading && items.length === 0
}
```

Search input: a controlled form using `useNavigate`; submit sets `?q=`. Group results by `authority` (Object.groupBy or reduce) and render each group under a heading.

- [ ] **Step 4: Point route** `/search` → `SearchPage` in `App.tsx`. Add a sidebar search link already present (Task 1).

- [ ] **Step 5: i18n keys** `search.title`, `search.placeholder`, `search.empty`, `search.results`, `search.evidence`, `search.incidents`, `search.documents`, `search.workflows` (en/vi).

- [ ] **Step 6: Run tests + lint + build; commit**

Run: `cd web && npx vitest run && npm run lint && npm run build`
Commit: `feat(web): hybrid search page with grouped evidence results`

---

### Task 11: Handover page transitions + final polish

**Files:**
- Modify: `web/src/console/pages/HandoverPage.tsx`
- Modify: `web/src/App.tsx`
- Test: extend `web/src/console/pages/HandoverPage.test.tsx`

**Interfaces:**
- Consumes: Task 3 (`HandoverPanel` moved), Task 2 (`api.submitHandover`, `api.acceptHandover`, `api.acknowledgeHandover`).
- Produces: `HandoverPage` with transition buttons gated by `handover:write` / `handover:accept`.

- [ ] **Step 1: Add tests for handover transitions**

Mock `api.handovers` returning a DRAFT handover. Assert: "Submit" button appears with `handover:write` permission; clicking calls `api.submitHandover`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/pages/HandoverPage.test.tsx`
Expected: FAIL.

- [ ] **Step 3: Wire transition buttons** in `HandoverPage`: DRAFT → Submit (`handover:write`), SUBMITTED → Accept (`handover:accept`), ACCEPTED → Acknowledge (`handover:accept`). Each calls the `api.*` method then `refetch()`. Error → toast.

- [ ] **Step 4: Run tests + lint + build**

Run: `cd web && npx vitest run && npm run lint && npm run build`
Expected: green.

- [ ] **Step 5: Commit**

```bash
git add web/src/console web/src/i18n/messages.ts
git commit -m "feat(web): handover page transitions and console polish"
```

---

### Task 12: Playwright verification pass

**Files:**
- Create: `web/scripts/verify-console.py` (playwright, mirrors the marketing verification pattern)

**Interfaces:**
- Consumes: the full routed console.
- Produces: screenshot + assertion evidence at desktop (1440) and mobile (390), both color schemes.

- [ ] **Step 1: Write the Playwright verification script**

```python
from playwright.sync_api import sync_playwright

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    for label, width, height in [("desktop", 1440, 900), ("mobile", 390, 844)]:
        page = browser.new_page(viewport={"width": width, "height": height})
        page.goto("http://localhost:5173/")
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(1000)
        overflow = page.evaluate("document.documentElement.scrollWidth > document.documentElement.clientWidth")
        print(f"[{label}] horizontal overflow: {overflow}")
        # Navigate to a detail page via sidebar
        page.get_by_text("Incident execution").first.click()
        page.wait_for_timeout(800)
        print(f"[{label}] incident page h1:", page.locator("h1").first.inner_text()[:40])
        page.screenshot(path=f"/tmp/console-{label}.png", full_page=True)
        page.close()
    browser.close()
```

- [ ] **Step 2: Run against the dev server**

Run: `python web/scripts/verify-console.py` with `npm run dev` running (use the webapp-testing `with_server.py` helper).
Expected: no horizontal overflow on either viewport; incident page navigates; screenshots saved.

- [ ] **Step 3: Verify role gating** — log in as `dev.supervisor` and as `dev.technician` (existing dev realm accounts); assert Reports nav item is absent for Technician (no `report:write`). Record the result in the task notes.

- [ ] **Step 4: Commit**

```bash
git add web/scripts/verify-console.py
git commit -m "test(web): add Playwright console verification pass"
```

---

## Self-Review

- **Spec coverage:** routes match the spec map (Task 2); dashboard (Task 4); assets (Task 5); incidents (Task 6); execution workbench (Task 7); reports (Task 8); document detail (Task 9); search (Task 10); handover (Task 11); shell/breadcrumbs/role nav (Task 1); useApi + states (Task 2); Playwright + role gating (Task 12). Quality/demonstrations/workflows preserved as pages (Task 3). Out-of-scope items (cluster B, marketing) untouched.
- **Search scope honesty:** spec said "grouped by type (incidents/documents/workflows)"; the API `/search` returns knowledge `Evidence` only, so Task 10 implements tabs with client-side filtering and documents the deviation.
- **Placeholder scan:** no TBD/TODO; every code step shows real code or a precise move instruction with exact component names.
- **Type consistency:** `useApi<T>` signature, `ShortcutCard` props, `StepRail` props, page component names, and `api.*` method names are identical across tasks. `draftReport` return type is checked in Task 7 step 4 before navigation wiring.

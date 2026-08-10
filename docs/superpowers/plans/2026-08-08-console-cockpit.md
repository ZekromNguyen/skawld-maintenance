# Console Cockpit (Linear-class, Light-first) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Re-express the Skawld console shell, dashboard, incidents, and execution workbench as a Linear-class cockpit: persistent global chrome, an annunciator-strip dashboard signature, alarm-log incident queues with saved views, metadata rails on detail pages, and a LOTO gate readout on the workbench. Light-first, emerald brand, Geist type, no new palette.

**Architecture:** Pure frontend. New primitives (`AnnunciatorStrip`, `FilterChip`, `SavedViews`, `MetadataRail`) compose existing pages; chrome moves from per-page `Topbar`/`PageHeader` into a persistent `GlobalBar` in `ConsoleLayout`; CSS tokens already exist in `styles.css` and are extended, not replaced. Data flow, routing, API, and state hooks are untouched.

**Tech Stack:** React 19, react-router 8, Vitest + Testing Library, Phosphor icons, plain CSS (`styles.css`), i18n via `messages.ts` (en + vi).

## Global Constraints

- No new accent color: emerald `--brand` is the only accent (design system §2.1).
- Numerals in counts/measurements use `font-variant-numeric: tabular-nums`.
- Motion only `transform`/`opacity`; collapsed under `prefers-reduced-motion`.
- Status never conveyed by color alone (badges carry text).
- i18n: every new label added to BOTH `en` and `vi` in `web/src/i18n/messages.ts` (same `MessageKey`).
- No em dashes in copy; sentence case; active voice.
- One `h1` per page; `:focus-visible` rings preserved.
- Do not modify API, routing, or state hooks.
- Existing tests must pass; extend tests where structure changes.

---

### Task 1: Global chrome — GlobalBar + slim PageHeader + Sidebar cleanup

**Files:**
- Create: `web/src/console/layout/GlobalBar.tsx`
- Modify: `web/src/console/layout/ConsoleLayout.tsx` (whole file)
- Modify: `web/src/console/layout/PageHeader.tsx:29-58` (remove search, site switcher, operator)
- Modify: `web/src/console/layout/Sidebar.tsx:106-131` (remove theme toggle + lang switch from footer)
- Modify: `web/src/styles.css` (GlobalBar styles; append near line 456)
- Test: `web/src/console/layout/GlobalBar.test.tsx` (new)

**Interfaces:**
- Consumes: `usePrincipal` (`Principal | undefined`), `useI18n` (`t`, `locale`, `setLocale`), `useTheme` (`theme`, `toggleTheme`), `SiteSwitcher` (`siteIds`, `value`, `onChange`), `useSite` (`siteId`, `setSiteId`), `CommandPalette` (existing, opened via `setOpen`).
- Produces: `GlobalBar` component rendered once by `ConsoleLayout` above `<Outlet>`; `PageHeader` without global chrome; `Sidebar` footer without theme/lang.

**Step 1: Write the failing test**

`web/src/console/layout/GlobalBar.test.tsx`:

```tsx
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { GlobalBar } from "./GlobalBar";
import { I18nProvider } from "../../i18n/I18nProvider";
import { ThemeProvider } from "../../theme/ThemeProvider";
import { SiteProvider } from "../state/SiteContext";

vi.mock("../../api", () => ({ api: {} }));

function renderBar() {
  return render(
    <ThemeProvider>
      <I18nProvider>
        <SiteProvider principal={{ id: "p1", display_name: "Tester", organization_id: "o1", site_ids: ["s1"], permissions: [] }}>
          <MemoryRouter initialEntries={["/"]}>
            <Routes>
              <Route path="*" element={<GlobalBar />} />
            </Routes>
          </MemoryRouter>
        </SiteProvider>
      </I18nProvider>
    </ThemeProvider>,
  );
}

describe("GlobalBar", () => {
  it("renders operator identity and search", () => {
    renderBar();
    expect(screen.getByText("Tester")).toBeTruthy();
    expect(screen.getByLabelText("Search commands…")).toBeTruthy();
  });
  it("opens the command palette on Cmd+K", () => {
    renderBar();
    fireEvent.keyDown(window, { key: "k", metaKey: true });
    expect(screen.getByText("Commands")).toBeTruthy();
  });
});
```

Note: import `fireEvent` from `@testing-library/react`.

**Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/console/layout/GlobalBar.test.tsx`
Expected: FAIL — module not found.

**Step 3: Implement `GlobalBar.tsx`**

```tsx
import { useState } from "react";
import { useNavigate } from "react-router";
import { MagnifyingGlass, Moon, Sun } from "@phosphor-icons/react";
import { useI18n } from "../../i18n/I18nProvider";
import { useTheme } from "../../theme/ThemeProvider";
import { useSite } from "../state/SiteContext";
import { SiteSwitcher } from "./SiteSwitcher";
import { CommandPalette } from "../components/CommandPalette";
import type { Principal } from "../../types";
import type { Locale } from "../../i18n/messages";

export function GlobalBar({ principal }: { principal?: Principal }) {
  const { t, locale, setLocale } = useI18n();
  const { theme, toggleTheme } = useTheme();
  const { siteId, setSiteId } = useSite();
  const navigate = useNavigate();
  const [paletteOpen, setPaletteOpen] = useState(false);

  return (
    <>
      <header className="global-bar">
        <button
          type="button"
          className="global-command"
          onClick={() => setPaletteOpen(true)}
        >
          <MagnifyingGlass size={14} aria-hidden="true" />
          <span>{t("topbar.searchPlaceholder")}</span>
          <kbd>⌘K</kbd>
        </button>
        <div className="global-bar-right">
          <SiteSwitcher siteIds={principal?.site_ids ?? []} value={siteId} onChange={setSiteId} />
          <button
            type="button"
            className="global-icon-button"
            onClick={toggleTheme}
            aria-label={theme === "light" ? t("theme.switchToDark") : t("theme.switchToLight")}
          >
            {theme === "light" ? <Moon size={14} /> : <Sun size={14} />}
          </button>
          <label className="global-lang">
            <select value={locale} onChange={(e) => setLocale(e.target.value as Locale)}>
              <option value="en">EN</option>
              <option value="vi">VI</option>
            </select>
          </label>
          <div className="operator">
            <span className="presence" />
            <span>
              <strong>{principal?.display_name ?? t("topbar.connecting")}</strong>
              <small>{principal?.site_ids.length ? t("topbar.siteScope", { count: principal.site_ids.length }) : t("topbar.allSiteScope")}</small>
            </span>
          </div>
        </div>
      </header>
      <CommandPalette principal={principal} open={paletteOpen} onOpenChange={setPaletteOpen} />
    </>
  );
}
```

**Step 4: Modify `CommandPalette.tsx` to accept optional controlled open state (backward-compatible)**

The existing `CommandPalette.test.tsx` renders `<CommandPalette principal={...} />` without open props and expects the component to manage its own state. Keep that working: make `open`/`onOpenChange` optional and fall back to internal state when absent.

In `web/src/console/components/CommandPalette.tsx`:
- Change signature to:

```tsx
export function CommandPalette({
  principal,
  open: controlledOpen,
  onOpenChange,
}: {
  principal?: Principal;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}) {
  const [internalOpen, setInternalOpen] = useState(false);
  const open = controlledOpen ?? internalOpen;
  const setOpen = (next: boolean) => {
    setInternalOpen(next);
    onOpenChange?.(next);
  };
```

- The existing `const [open, setOpen] = useState(false)` becomes the internal state above; the `useEffect` Cmd+K keydown listener stays as-is but must call `setOpen((current) => !current)` via the wrapper (it already calls `setOpen`, which now also forwards to `onOpenChange`). Keep `preventDefault`.
- Keep the Radix `DialogPrimitive.Root open={open} onOpenChange={setOpen}` wiring (lines 85+).
- Existing tests pass unchanged (uncontrolled path); `GlobalBar` passes `open`/`onOpenChange` (controlled path).
- Update `ConsoleLayout.tsx` to render `GlobalBar` instead of `<CommandPalette>` (see Step 5).

**Step 5: Rewrite `ConsoleLayout.tsx`**

```tsx
import { Outlet } from "react-router";
import { Sidebar } from "./Sidebar";
import { GlobalBar } from "./GlobalBar";
import { usePrincipal } from "../usePrincipal";
import { SiteProvider } from "../state/SiteContext";

export function ConsoleLayout() {
  const { data: principal } = usePrincipal();
  return (
    <SiteProvider principal={principal}>
      <a href="#main-content" className="skip-link">
        Skip to content
      </a>
      <div className="app-shell">
        <Sidebar principal={principal} />
        <div className="app-main">
          <GlobalBar principal={principal} />
          <main id="main-content" style={{ minWidth: 0, padding: "0 28px 40px" }}>
            <Outlet />
          </main>
        </div>
      </div>
    </SiteProvider>
  );
}
```

**Step 6: Slim `PageHeader.tsx`**

Remove from `PageHeader.tsx:29-58`: the `SiteSwitcher`, the search `<form>`, and the `.operator` div. Keep breadcrumbs, title row, and `actions`. Drop unused imports (`SiteSwitcher`, `useSite`, `useNavigate` if now unused — the search form used `navigate`; `Breadcrumbs` stays). The operator/search/site now live in `GlobalBar`.

**Step 7: Clean `Sidebar.tsx` footer**

Remove the theme toggle button and the lang-switch label from `Sidebar.tsx:106-122`. Keep the safety boundary and logout button. Remove unused imports (`useTheme`, `Moon`, `Sun`, `Locale`, `setLocale` usage; keep `useI18n` for `t`).

**Step 8: Add GlobalBar CSS to `styles.css`**

Append after the `.page-header` block (near line 456):

```css
.global-bar { display: flex; align-items: center; justify-content: space-between; gap: 16px; min-height: 56px; padding: 0 28px; border-bottom: 1px solid var(--line); background: var(--surface); }
.global-command { display: inline-flex; align-items: center; gap: 8px; min-width: 260px; padding: 7px 10px; border: 1px solid var(--line-soft); border-radius: var(--radius-input); background: var(--sunken); color: var(--ink-muted); font-size: 12px; }
.global-command:hover { border-color: var(--brand-dim); color: var(--ink-soft); }
.global-command kbd { margin-left: auto; border: 1px solid var(--line); border-radius: 4px; padding: 1px 5px; font: 600 10px ui-monospace, monospace; color: var(--ink-muted); background: var(--raised); }
.global-bar-right { display: flex; align-items: center; gap: 12px; }
.global-icon-button { display: grid; place-items: center; width: 30px; height: 30px; border: 1px solid var(--line-soft); border-radius: var(--radius-button); background: transparent; color: var(--ink-soft); }
.global-icon-button:hover { background: var(--raised); color: var(--ink); }
.global-lang select { padding: 5px 7px; border: 1px solid var(--line-soft); border-radius: var(--radius-input); background: var(--raised); color: var(--ink); font-size: 11px; }
.app-main { min-width: 0; display: flex; flex-direction: column; }
@media (max-width: 720px) {
  .global-bar { padding: 0 12px; }
  .global-command { min-width: 0; flex: 1; }
  .global-bar-right .operator { display: none; }
}
```

Also update the existing `@media (max-width: 720px)` `.app-shell { display: block; }` rule stays; `.sidebar` mobile layout unchanged.

**Step 9: Run tests**

Run: `cd web && npx vitest run src/console/layout src/console/components/CommandPalette.test.tsx src/App.test.tsx 2>/dev/null; npm test 2>&1 | tail -5`
Expected: PASS for GlobalBar, ConsoleLayout, Sidebar; no new failures beyond the pre-existing KnowledgePage one.

**Step 10: Run lint**

Run: `cd web && npm run lint`
Expected: no type errors.

**Step 11: Commit**

```bash
git add web/src/console/layout/GlobalBar.tsx web/src/console/layout/ConsoleLayout.tsx web/src/console/layout/PageHeader.tsx web/src/console/layout/Sidebar.tsx web/src/console/components/CommandPalette.tsx web/src/console/layout/GlobalBar.test.tsx web/src/styles.css
git commit -m "feat: move console chrome into a persistent global bar"
```

---

### Task 2: AnnunciatorStrip dashboard signature

**Files:**
- Create: `web/src/console/components/AnnunciatorStrip.tsx`
- Create: `web/src/console/components/AnnunciatorStrip.test.tsx`
- Modify: `web/src/styles.css` (annunciator styles near `.metrics` at line 126)

**Interfaces:**
- Consumes: none external; props only.
- Produces: `<AnnunciatorStrip cells={Cell[]} />` where `Cell = { key, label, value, tone: "critical"|"high"|"medium"|"info"|"success", to?: string, eyebrow?: string }`.

**Step 1: Write the failing test**

`AnnunciatorStrip.test.tsx`:

```tsx
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { AnnunciatorStrip } from "./AnnunciatorStrip";

const cells = [
  { key: "open", label: "Open incidents", value: "12", tone: "critical" as const, to: "/incidents" },
  { key: "exec", label: "In progress", value: "3", tone: "medium" as const },
];

describe("AnnunciatorStrip", () => {
  it("renders a cell per item with tabular numerals", () => {
    render(
      <MemoryRouter>
        <AnnunciatorStrip cells={cells} />
      </MemoryRouter>,
    );
    expect(screen.getByText("Open incidents")).toBeTruthy();
    expect(screen.getByText("12")).toBeTruthy();
  });
  it("wraps cells with a link when to is set", () => {
    render(
      <MemoryRouter>
        <AnnunciatorStrip cells={cells} />
      </MemoryRouter>,
    );
    const link = screen.getByRole("link", { name: /Open incidents/ });
    expect(link.getAttribute("href")).toBe("/incidents");
  });
  it("renders a truthful zero", () => {
    render(
      <MemoryRouter>
        <AnnunciatorStrip cells={[{ key: "z", label: "Pending", value: "0", tone: "info" }]} />
      </MemoryRouter>,
    );
    expect(screen.getByText("0")).toBeTruthy();
  });
});
```

**Step 2: Run to verify it fails**

Run: `cd web && npx vitest run src/console/components/AnnunciatorStrip.test.tsx`
Expected: FAIL — module not found.

**Step 3: Implement `AnnunciatorStrip.tsx`**

```tsx
import { Link } from "react-router";
import type { Tone } from "../ui/StatusBadge";

export interface AnnunciatorCell {
  key: string;
  label: string;
  value: string;
  tone: Tone;
  to?: string;
  eyebrow?: string;
}

export function AnnunciatorStrip({ cells }: { cells: AnnunciatorCell[] }) {
  return (
    <section className="annunciator" role="list" aria-label="Operational status">
      {cells.map((cell) => {
        const body = (
          <div className={`annunciator-cell annunciator-${cell.tone}`}>
            <span className="annunciator-label">{cell.label}</span>
            <strong className="mono-num">{cell.value}</strong>
            {cell.eyebrow ? <small className="annunciator-eyebrow">{cell.eyebrow}</small> : null}
          </div>
        );
        return (
          <div key={cell.key} role="listitem">
            {cell.to ? (
              <Link to={cell.to} className="annunciator-link" aria-label={cell.label}>
                {body}
              </Link>
            ) : (
              body
            )}
          </div>
        );
      })}
    </section>
  );
}
```

**Step 4: Add CSS**

Near line 126 (`.metrics` block), append:

```css
.annunciator { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin-bottom: 20px; }
.annunciator-cell { min-height: 96px; display: flex; flex-direction: column; justify-content: space-between; padding: 14px 16px 13px; border: 1px solid var(--line); border-left: 3px solid var(--ink-muted); background: var(--surface); }
.annunciator-link { text-decoration: none; color: inherit; display: block; }
.annunciator-link:hover .annunciator-cell { border-color: var(--brand-dim); }
.annunciator-critical { border-left-color: var(--red); }
.annunciator-high { border-left-color: var(--amber); }
.annunciator-medium { border-left-color: var(--amber); }
.annunciator-info { border-left-color: var(--blue); }
.annunciator-success { border-left-color: var(--brand); }
.annunciator-label { color: var(--ink-muted); font-size: 9px; font-weight: 700; text-transform: uppercase; letter-spacing: .08em; }
.annunciator-cell strong { font: 750 32px/1.1 ui-monospace, monospace; color: var(--ink); margin-top: 6px; }
.annunciator-eyebrow { color: var(--ink-muted); font-size: 10px; margin-top: 4px; }
.mono-num { font-variant-numeric: tabular-nums; }
@media (max-width: 1050px) { .annunciator { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 720px) { .annunciator { grid-template-columns: 1fr 1fr; } }
```

Note: `font: 750 32px/1.1 ui-monospace` keeps the existing mono stack; add `font-variant-numeric` via the `.mono-num` class on `strong`.

**Step 5: Run tests**

Run: `cd web && npx vitest run src/console/components/AnnunciatorStrip.test.tsx`
Expected: PASS.

**Step 6: Commit**

```bash
git add web/src/console/components/AnnunciatorStrip.tsx web/src/console/components/AnnunciatorStrip.test.tsx web/src/styles.css
git commit -m "feat: add annunciator strip dashboard signature"
```

---

### Task 3: Wire AnnunciatorStrip into the dashboard

**Files:**
- Modify: `web/src/console/components/Overview.tsx:57-84` (replace `.metrics` section)
- Modify: `web/src/console/pages/DashboardPage.test.tsx` (assert annunciator labels still render; keep existing assertions)
- Modify: `web/src/i18n/messages.ts` (add `annunciator.eyebrows` keys if used; reuse existing `dashboard.*` keys where possible)

**Interfaces:**
- Consumes: `AnnunciatorStrip`, `AnnunciatorCell` (Task 2), existing `focus`, `incidents`, `executions`, `assets`, `pendingHandoverCount` props.
- Produces: dashboard renders `AnnunciatorStrip` with 4 cells (open incidents → `/incidents` critical; in-progress executions → `/executions` medium; critical assets → `/assets` high; pending handovers → `/handovers` info).

**Step 1: Modify `Overview.tsx`**

Replace the `<section className="metrics">...</section>` block (`Overview.tsx:59-84`) with:

```tsx
<AnnunciatorStrip
  cells={[
    { key: "open-incidents", label: t("dashboard.openIncidents"), value: String(open.length), tone: "critical", to: "/incidents" },
    { key: "in-progress", label: t("dashboard.executionsInProgress"), value: String(inProgress.length), tone: "medium", to: "/executions" },
    { key: "critical-assets", label: t("dashboard.criticalAssets"), value: String(critical.length), tone: "high", to: "/assets" },
    { key: "pending-handovers", label: t("dashboard.pendingHandovers"), value: String(pendingHandoverCount), tone: "info", to: "/handovers" },
  ]}
/>
```

Add `import { AnnunciatorStrip } from "./AnnunciatorStrip";`. Keep the rest of `Overview` (focus panel, active-incident table) unchanged. `MetricCard` import can be removed if no longer used elsewhere in the file (check first).

**Step 2: Update DashboardPage.test.tsx**

Existing assertions use `dashboard.openIncidents` etc. via `getByText`. Verify the test still finds the labels (it should, since the label text is unchanged). Add one assertion that the strip link exists:

```tsx
expect(screen.getByRole("link", { name: /Open incidents/i })).toBeTruthy();
```

**Step 3: Run tests**

Run: `cd web && npx vitest run src/console/pages/DashboardPage.test.tsx`
Expected: PASS.

**Step 4: Commit**

```bash
git add web/src/console/components/Overview.tsx web/src/console/pages/DashboardPage.test.tsx
git commit -m "feat: show dashboard KPIs as an annunciator strip"
```

---

### Task 4: Incident alarm-log rows + severity edge on board cards

**Files:**
- Modify: `web/src/console/pages/IncidentsPage.tsx:191-243` (list rendering: add severity edge to each row)
- Modify: `web/src/console/components/IncidentBoard.tsx:70-82` (BoardCard: add top severity edge stripe)
- Modify: `web/src/styles.css` (`.alarm-row`, `.alarm-edge` styles near `.board-*` at line 521)
- Test: `web/src/console/pages/IncidentsPage.test.tsx` (extend: alarm edge class present)

**Interfaces:**
- Consumes: existing `severityTone`/`severityLabelKey`; new CSS classes `.alarm-edge--{critical|high|medium|low}`.
- Produces: list rows wrapped in `<div className="alarm-row">` with a severity edge `<span>`; board cards get `className="board-card board-card--{sev}"`.

**Step 1: Implement list alarm edge**

The current list uses `DataTable<Incident>`. DataTable renders `<td>` per column, so a full-height left edge per row needs the edge inside the first cell or a `tr` class. Simplest robust approach: extend `DataTable` with an optional `rowClassName?: (row: T) => string` prop applied to `<tr>` (check `DataTable.tsx` for the `<tr>` render location, lines 58-130) and add a first-column severity edge via a new leading column:

In `DataTable.tsx`, add to the `<tr>`:

```tsx
<tr key={key} className={rowClassName?.(row)}>
```

and to the props interface: `rowClassName?: (row: T) => string;`.

In `IncidentsPage.tsx`, pass:

```tsx
rowClassName={(incident) => `alarm-row alarm-row--${incident.severity.toLowerCase()}`}
```

and prepend a first column:

```tsx
{
  key: "edge",
  header: "",
  render: () => <span className="alarm-edge" aria-hidden="true" />,
},
```

**Step 2: Board card edge**

In `IncidentBoard.tsx`, change BoardCard root:

```tsx
<Link to={`/incidents/${incident.id}`} className={`board-card board-card--${incident.severity.toLowerCase()}`}>
```

**Step 3: CSS**

Append near the board block (line 521):

```css
.alarm-row td:first-child { width: 3px; padding: 0; }
.alarm-edge { display: block; width: 3px; height: 100%; min-height: 44px; background: var(--ink-muted); }
.alarm-row--critical .alarm-edge { background: var(--red); }
.alarm-row--high .alarm-edge { background: var(--amber); }
.alarm-row--medium .alarm-edge { background: var(--amber); }
.alarm-row--low .alarm-edge { background: var(--brand); }
.board-card { border-top: 3px solid var(--line-soft); }
.board-card--critical { border-top-color: var(--red); }
.board-card--high { border-top-color: var(--amber); }
.board-card--medium { border-top-color: var(--amber); }
.board-card--low { border-top-color: var(--brand); }
```

**Step 4: Update IncidentsPage.test.tsx**

Add:

```tsx
it("renders alarm-row severity edges in list view", async () => {
  // reuse existing render; assert rowClassName applied via a row containing the HIGH incident
  const row = screen.getByRole("row", { name: /Pump vibration/ });
  expect(row.className).toContain("alarm-row--high");
});
```

**Step 5: Run tests**

Run: `cd web && npx vitest run src/console/pages/IncidentsPage.test.tsx src/console/components/IncidentBoard.test.tsx 2>/dev/null || npx vitest run src/console/pages/IncidentsPage.test.tsx`
Expected: PASS.

**Step 6: Commit**

```bash
git add web/src/console/pages/IncidentsPage.tsx web/src/console/components/IncidentBoard.tsx web/src/console/ui/DataTable.tsx web/src/console/pages/IncidentsPage.test.tsx web/src/styles.css
git commit -m "feat: add severity edge language to incident lists and boards"
```

---

### Task 5: Incident saved views + filter chips

**Files:**
- Create: `web/src/console/components/SavedViews.tsx`
- Create: `web/src/console/components/SavedViews.test.tsx`
- Modify: `web/src/console/pages/IncidentsPage.tsx:123-181` (toolbar: add chip row + saved views)
- Modify: `web/src/styles.css` (chip + saved-view styles near `.incident-toolbar` at line 467)
- Modify: `web/src/i18n/messages.ts` (add `incidents.views.*` keys en + vi)

**Interfaces:**
- Consumes: `useI18n`; `SavedView = { id: string; name: string; severity: string; query: string }`.
- Produces: `<SavedViews views={...} activeId onSave onApply onDelete />`; `FilterChip` inline component (or `.chip` CSS class used by both).

**Step 1: Write the failing test**

`SavedViews.test.tsx`:

```tsx
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SavedViews } from "./SavedViews";
import { I18nProvider } from "../../i18n/I18nProvider";

const views = [{ id: "v1", name: "My critical queue", severity: "CRITICAL", query: "" }];

function renderViews(props: { onSave: () => void; onApply: (id: string) => void; onDelete: (id: string) => void }) {
  return render(
    <I18nProvider>
      <SavedViews views={views} activeId={null} onSave={props.onSave} onApply={props.onApply} onDelete={props.onDelete} />
    </I18nProvider>,
  );
}

describe("SavedViews", () => {
  it("lists saved views and applies on click", () => {
    const onApply = vi.fn();
    renderViews({ onSave: vi.fn(), onApply, onDelete: vi.fn() });
    fireEvent.click(screen.getByText("My critical queue"));
    expect(onApply).toHaveBeenCalledWith("v1");
  });
  it("offers save current filter", () => {
    const onSave = vi.fn();
    renderViews({ onSave, onApply: vi.fn(), onDelete: vi.fn() });
    fireEvent.click(screen.getByRole("button", { name: /Save view/ }));
    expect(onSave).toHaveBeenCalled();
  });
});
```

**Step 2: Run to verify it fails**

Run: `cd web && npx vitest run src/console/components/SavedViews.test.tsx`
Expected: FAIL — module not found.

**Step 3: Implement `SavedViews.tsx`**

```tsx
import { BookmarkSimple, FloppyDisk, Trash } from "@phosphor-icons/react";
import { useI18n } from "../../i18n/I18nProvider";

export interface SavedView {
  id: string;
  name: string;
  severity: string;
  query: string;
}

export function SavedViews({
  views,
  activeId,
  onSave,
  onApply,
  onDelete,
}: {
  views: SavedView[];
  activeId: string | null;
  onSave: () => void;
  onApply: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  const { t } = useI18n();
  return (
    <div className="saved-views" role="group" aria-label={t("incidents.views.label")}>
      <button type="button" className="chip chip-action" onClick={onSave} title={t("incidents.views.save")}>
        <FloppyDisk size={12} aria-hidden="true" />
        {t("incidents.views.save")}
      </button>
      {views.map((view) => (
        <span key={view.id} className={`chip${view.id === activeId ? " active" : ""}`}>
          <button type="button" className="chip-apply" onClick={() => onApply(view.id)}>
            <BookmarkSimple size={12} aria-hidden="true" />
            {view.name}
          </button>
          <button
            type="button"
            className="chip-delete"
            aria-label={`${t("incidents.views.delete")}: ${view.name}`}
            onClick={() => onDelete(view.id)}
          >
            <Trash size={11} aria-hidden="true" />
          </button>
        </span>
      ))}
    </div>
  );
}
```

**Step 4: Add CSS near `.incident-toolbar` (line 467)**

```css
.chip { display: inline-flex; align-items: center; gap: 6px; border: 1px solid var(--line); border-radius: 999px; background: var(--surface); padding: 5px 10px; font-size: 11px; font-weight: 600; color: var(--ink-soft); }
.chip:hover { border-color: var(--brand-dim); }
.chip.active { border-color: var(--brand-dim); color: var(--brand); background: color-mix(in srgb, var(--brand) 8%, var(--surface)); }
.chip-action { border-style: dashed; }
.chip-apply { border: 0; background: transparent; color: inherit; font: inherit; display: inline-flex; align-items: center; gap: 6px; padding: 0; cursor: pointer; }
.chip-delete { border: 0; background: transparent; color: var(--ink-muted); padding: 2px; cursor: pointer; display: inline-flex; }
.chip-delete:hover { color: var(--red); }
.saved-views { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 12px; }
```

**Step 5: Wire into `IncidentsPage.tsx`**

- Add state: `const [savedViews, setSavedViews] = useState<SavedView[]>(() => readSavedViews())` with a `SAVED_VIEWS_KEY = "skawld.incidents.savedViews"` localStorage helper (try/catch, mirroring `initialView`).
- Render `<SavedViews views={savedViews} activeId={...} onSave={saveCurrent} onApply={applyView} onDelete={deleteView} />` above the `.incident-toolbar` div.
- `saveCurrent`: push `{ id: crypto.randomUUID?.() ?? String(Date.now()), name: t("incidents.views.defaultName"), severity, query }` when severity or query differ from ALL/""; persist.
- `applyView(id)`: set severity/query from the stored view; set `activeId`.
- `deleteView(id)`: filter out; persist.
- Default seed when storage empty: `[{ id: "my-critical", name: t("incidents.views.myCritical"), severity: "CRITICAL", query: "" }]` — only if storage is empty (do not overwrite user data).

**Step 6: i18n keys (en + vi)**

Add to `en` (and matching `vi` entries in the same positions):

```ts
"incidents.views.label": "Saved views",
"incidents.views.save": "Save view",
"incidents.views.delete": "Delete view",
"incidents.views.defaultName": "Saved filter",
"incidents.views.myCritical": "My critical queue",
```

Vietnamese:
```ts
"incidents.views.label": "Chế độ xem đã lưu",
"incidents.views.save": "Lưu chế độ xem",
"incidents.views.delete": "Xóa chế độ xem",
"incidents.views.defaultName": "Bộ lọc đã lưu",
"incidents.views.myCritical": "Hàng đợi nghiêm trọng của tôi",
```

**Step 7: Run tests**

Run: `cd web && npx vitest run src/console/components/SavedViews.test.tsx src/console/pages/IncidentsPage.test.tsx`
Expected: PASS.

**Step 8: Commit**

```bash
git add web/src/console/components/SavedViews.tsx web/src/console/components/SavedViews.test.tsx web/src/console/pages/IncidentsPage.tsx web/src/i18n/messages.ts web/src/styles.css
git commit -m "feat: add saved views and filter chips to the incident queue"
```

---

### Task 6: Incident detail metadata rail

**Files:**
- Modify: `web/src/console/pages/IncidentDetailPage.tsx:143-168` (wrap facts panel into rail layout)
- Modify: `web/src/styles.css` (`.detail-layout`, `.metadata-rail` styles near `.incident-facts` at line 482)
- Test: `web/src/console/pages/IncidentDetailPage.test.tsx` (extend: rail present)

**Interfaces:**
- Consumes: existing incident data, `StatusBadge`, `RelativeTime`.
- Produces: `.detail-layout` grid (main content 1fr, `.metadata-rail` 260px) wrapping the facts panel on the right.

**Step 1: Modify `IncidentDetailPage.tsx`**

Wrap the current single-column `<div style={{ display: "grid", gap: 16 }}>` (line 143) content:

```tsx
<div className="detail-layout">
  <div className="detail-main" style={{ display: "grid", gap: 16 }}>
    {/* existing panels: incidents table, recommendation, etc. */}
  </div>
  <aside className="metadata-rail">
    <div className="panel">
      <div className="panel-heading"><h2>{t("incident.facts")}</h2></div>
      <div className="incident-facts">
        {/* existing fact blocks: asset, severity, state, detected */}
      </div>
    </div>
  </aside>
</div>
```

Keep all existing content; only the layout wrapper changes. Move the facts panel (lines 144-168) into the rail.

**Step 2: CSS**

Append near `.incident-facts` (line 482):

```css
.detail-layout { display: grid; grid-template-columns: minmax(0, 1fr) 260px; gap: 16px; align-items: start; }
.metadata-rail { display: grid; gap: 16px; position: sticky; top: 72px; }
.detail-layout .incident-facts { grid-template-columns: 1fr; gap: 14px; padding: 14px 16px; }
@media (max-width: 1050px) { .detail-layout { grid-template-columns: 1fr; } .metadata-rail { position: static; } }
```

**Step 3: Extend IncidentDetailPage.test.tsx**

Add:

```tsx
it("renders the metadata rail", async () => {
  // after existing render + wait
  expect(screen.getByRole("complementary")).toBeTruthy();
});
```

**Step 4: Run tests**

Run: `cd web && npx vitest run src/console/pages/IncidentDetailPage.test.tsx`
Expected: PASS.

**Step 5: Commit**

```bash
git add web/src/console/pages/IncidentDetailPage.tsx web/src/console/pages/IncidentDetailPage.test.tsx web/src/styles.css
git commit -m "feat: add metadata rail to incident detail"
```

---

### Task 7: Workbench LOTO gate readout + metadata rail

**Files:**
- Modify: `web/src/console/pages/ExecutionWorkbenchPage.tsx:156-224` (LOTO banner above StepRail; wrap right column into rail)
- Modify: `web/src/styles.css` (`.loto-banner` styles near `.step-rail` at line 491)
- Test: `web/src/console/pages/ExecutionWorkbenchPage.test.tsx` (extend: banner appears only when a blocked step exists)

**Interfaces:**
- Consumes: `value.steps` (each `Step` has `state`, `blocked_reason`, `risk_level`), `useI18n`.
- Produces: `.loto-banner` rendered only when at least one step is `BLOCKED`.

**Step 1: Modify `ExecutionWorkbenchPage.tsx`**

Before `<StepRail ... />` (line 157), insert:

```tsx
{value.steps.some((step) => step.state === "BLOCKED") && (
  <div className="loto-banner" role="status">
    <span className="loto-banner-dot" aria-hidden="true" />
    <div>
      <strong>{t("workbench.lotoActive")}</strong>
      <p>{t("workbench.lotoBody")}</p>
    </div>
  </div>
)}
```

Wrap the right column (measurements/observations/evidence panels, lines 158-223) in `<div className="workbench-rail">` and keep the existing two-panel structure inside `detail-layout`-like grid: change the outer `<div style={{ display: "grid", gap: 16 }}>` (line 156) to `<div className="detail-layout">`, and wrap StepRail in `<div className="detail-main">` while the right column becomes the rail. If StepRail stays left and full-height, use:

```tsx
<div className="workbench-layout">
  <div className="workbench-steps">
    {banner}
    <StepRail ... />
  </div>
  <div className="workbench-rail"> {/* measurements, observations, evidence */} </div>
</div>
```

**Step 2: CSS**

```css
.workbench-layout { display: grid; grid-template-columns: minmax(0, 1.25fr) minmax(0, 1fr); gap: 16px; align-items: start; }
.workbench-steps, .workbench-rail { display: grid; gap: 16px; }
.loto-banner { display: flex; align-items: flex-start; gap: 12px; padding: 12px 16px; border: 1px solid var(--red); border-left: 3px solid var(--red); background: color-mix(in srgb, var(--red) 7%, var(--surface)); }
.loto-banner-dot { width: 10px; height: 10px; border-radius: 50%; background: var(--red); margin-top: 3px; flex-shrink: 0; }
.loto-banner strong { display: block; font: 750 12px ui-monospace, monospace; color: var(--red); letter-spacing: .06em; }
.loto-banner p { margin: 4px 0 0; color: var(--ink-soft); font-size: 11px; line-height: 1.5; }
@media (max-width: 1050px) { .workbench-layout { grid-template-columns: 1fr; } }
```

**Step 3: i18n keys**

```ts
"workbench.lotoActive": "LOTO ACTIVE - intrusive step gated",
"workbench.lotoBody": "A locked-out step is blocking progress. Clear the gate before continuing.",
```

Vietnamese:
```ts
"workbench.lotoActive": "LOTO ĐANG KÍCH HOẠT - bước xâm nhập bị khóa",
"workbench.lotoBody": "Một bước khóa an toàn đang chặn tiến độ. Hãy giải phóng khóa trước khi tiếp tục.",
```

**Step 4: Extend ExecutionWorkbenchPage.test.tsx**

Find the existing test fixture; add a case with a BLOCKED step and assert:

```tsx
expect(screen.getByText(/LOTO ACTIVE/)).toBeTruthy();
```

and a case without blocked steps asserting `queryByText(/LOTO ACTIVE/)` is null.

**Step 5: Run tests**

Run: `cd web && npx vitest run src/console/pages/ExecutionWorkbenchPage.test.tsx`
Expected: PASS.

**Step 6: Commit**

```bash
git add web/src/console/pages/ExecutionWorkbenchPage.tsx web/src/console/pages/ExecutionWorkbenchPage.test.tsx web/src/i18n/messages.ts web/src/styles.css
git commit -m "feat: surface LOTO gate state on the execution workbench"
```

---

### Task 8: Full verification

**Step 1: Full test suite**

Run: `cd web && npm test 2>&1 | tail -6`
Expected: 1 pre-existing failure only (KnowledgePage.test.tsx); everything else PASS.

**Step 2: Lint**

Run: `cd web && npm run lint`
Expected: no errors.

**Step 3: E2E smoke (if backend fixtures available)**

Run: `cd web && npx playwright test 2>&1 | tail -8`
If the e2e suite requires a running API, note the result rather than fixing infrastructure.

**Step 4: Build**

Run: `cd web && npm run build`
Expected: success.

**Step 5: Manual checks**

- Light + dark theme toggle from GlobalBar.
- en + vi language switch from GlobalBar.
- Keyboard-only nav (Tab through GlobalBar buttons, focus rings visible).
- Reduced motion: annunciator/transition CSS collapsed (no animations added, so verify no new motion exists).

**Step 6: Report**

Summarize: what changed per file, test results, the pre-existing KnowledgePage failure (unrelated, left untouched), and the e2e/build outcome.

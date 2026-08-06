# Role × Screen Matrix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enforce the role × screen matrix (spec `2026-08-06-role-screen-matrix-design.md`) in code: lock the permission sets with a Go test, add a web permission manifest + gate, wire nav gating, and make the dashboard role-aware.

**Architecture:** The backend permission mapping (`PermissionsForRole`) is already the source of truth; a characterization test locks it to the matrix. On the web, a pure `permissions.ts` manifest (permission keys, `can()`, `focusRole()`) feeds a `PermissionGate` component, the Sidebar's existing per-item gating, and a new role-focus panel on the dashboard. Focus is derived from permissions, never role names (per the IA principle "gating by permission, not role name").

**Tech Stack:** Go 1.25 (`slices`, `testing`), React 19 + react-router 8, Vitest 4 + Testing Library, TypeScript strict, existing token-driven `styles.css`.

## Global Constraints

- Gating is by **permission, not role name** (pilot console IA).
- Cluster B screens (users & roles, orgs & sites, settings, integrations, audit) are **out of scope**; the manifest includes their permission keys but no screens.
- Every new UI string gets **en + vi** keys; the `MessageKey`/`Record` typing in `web/src/i18n/messages.ts` enforces this (compile error if a key is missing in either locale).
- Console stays dark-first, Geist type, existing classes (`panel`, `metrics`, `secondary-button`); no new styling dependencies.
- TDD: write the failing test, verify it fails (or locks), implement, verify green, commit per task.

## File Structure

| File | Responsibility |
|---|---|
| `internal/identity/domain/authorization_test.go` | Lock the exact permission set per role to the matrix (characterization test) |
| `web/src/console/permissions.ts` | New: `PermissionKey` union, `hasPermission`, `can`, `RoleFocus`, `focusRole` — pure, no React |
| `web/src/console/permissions.test.ts` | New: tests for the manifest functions |
| `web/src/console/ui/PermissionGate.tsx` | New: renders children only when the principal has the required permission(s) |
| `web/src/console/ui/PermissionGate.test.tsx` | New: gate rendering tests |
| `web/src/console/layout/Sidebar.tsx` | Modify: use `can()` from the manifest for nav-item gating; `NavItem.permission` typed as `PermissionKey` |
| `web/src/console/layout/Sidebar.test.tsx` | Modify: assert Reports visible for technician, hidden for manager |
| `web/src/console/dashboardFocus.ts` | New: `FOCUS_CONFIG` map `RoleFocus → {titleKey, links}` |
| `web/src/console/pages/DashboardPage.tsx` | Modify: compute `focusRole(principal)` and pass the focus config to `Overview` |
| `web/src/console/components/Overview.tsx` | Modify: render the focus panel above "My queue" |
| `web/src/console/pages/DashboardPage.test.tsx` | Modify: assert focus panels for admin/supervisor/manager principals |
| `web/src/i18n/messages.ts` | Modify: add `dashboard.focus.*` keys (en + vi) |

---

### Task 1: Lock the role → permission matrix in Go

**Files:**
- Modify: `internal/identity/domain/authorization_test.go` (append after line 143)

**Interfaces:**
- Consumes: `PermissionsForRole(role)` from `authorization.go:93`.
- Produces: `TestPermissionsForRoleMatrix` — the exact expected set per role, sorted, as `[]string`.

- [ ] **Step 1: Write the failing (locks) test**

Append to `internal/identity/domain/authorization_test.go`:

```go
func TestPermissionsForRoleMatrix(t *testing.T) {
	t.Parallel()
	read := []string{
		"asset:read", "demonstration:read", "execution:read",
		"incident:read", "knowledge:read", "workflow:read",
	}
	expected := map[Role][]string{
		RoleAdministrator: append(slices.Clone(read),
			"asset:create", "asset:criticality:approve", "attachment:write",
			"demonstration:capture", "demonstration:review", "execution:read:all",
			"execution:write", "handover:accept", "handover:write",
			"incident:create", "incident:resolve", "integration:external:import",
			"knowledge:approve", "knowledge:write", "organization:create",
			"prerequisite:verify",
			"recommendation:review", "recommendation:run", "report:approve",
			"report:write", "workflow:publish", "workflow:review",
		),
		RoleMaintenanceSupervisor: append(slices.Clone(read),
			"asset:create", "asset:criticality:approve", "attachment:write",
			"demonstration:capture", "demonstration:review", "execution:read:all",
			"execution:write", "handover:accept", "handover:write",
			"incident:create", "incident:resolve", "integration:external:import",
			"knowledge:approve", "knowledge:write", "prerequisite:verify",
			"recommendation:review", "recommendation:run", "report:approve",
			"report:write", "workflow:review",
		),
		RoleSeniorTechnician: append(slices.Clone(read),
			"attachment:write", "demonstration:capture", "demonstration:review",
			"execution:write", "handover:write", "incident:create",
			"prerequisite:verify", "recommendation:run", "report:write",
			"workflow:review",
		),
		RoleTechnician: append(slices.Clone(read),
			"attachment:write", "demonstration:capture", "execution:write",
			"handover:write", "recommendation:run", "report:write",
		),
		RoleManager: append(slices.Clone(read),
			"demonstration:review", "handover:accept", "handover:write",
			"recommendation:review", "recommendation:run",
		),
	}
	for role, want := range expected {
		got := permissionSet(PermissionsForRole(role))
		if !slices.Equal(got, want) {
			t.Errorf("PermissionsForRole(%s) = %v, want %v", role, got, want)
		}
	}
}

// permissionSet returns the role's permissions as a sorted, deduplicated
// list of strings, so expected sets can be written literally.
func permissionSet(permissions []Permission) []string {
	unique := make(map[Permission]struct{}, len(permissions))
	for _, p := range permissions {
		unique[p] = struct{}{}
	}
	set := make([]string, 0, len(unique))
	for p := range unique {
		set = append(set, string(p))
	}
	sort.Strings(set)
	return set
}
```

Add imports to the test file header:

```go
import (
	"slices"
	"sort"
	"testing"
	"time"
)
```

- [ ] **Step 2: Run the test**

Run: `go test ./internal/identity/domain/ -run TestPermissionsForRoleMatrix -v`
Expected: PASS — the current `PermissionsForRole` already matches the matrix (this is a characterization lock; `permissionSet` dedupes the overlap between `read` and the role additions, e.g. none today). If it FAILS, the diff lists the offending permission strings; fix `PermissionsForRole` in `authorization.go` so the sets match the test exactly (that is the matrix), then re-run to green.

- [ ] **Step 3: Run the full package test suite**

Run: `go test ./internal/identity/domain/`
Expected: all PASS (existing tests unchanged).

- [ ] **Step 4: Commit**

```bash
git add internal/identity/domain/authorization_test.go
git commit -m "test: lock role permission matrix to the role-screen spec"
```

---

### Task 2: Web permission manifest

**Files:**
- Create: `web/src/console/permissions.ts`
- Test: `web/src/console/permissions.test.ts`

**Interfaces:**
- Consumes: `Principal` type from `web/src/types.ts`.
- Produces:
  - `type PermissionKey` — union of every permission the UI checks.
  - `hasPermission(principal: Pick<Principal, "permissions"> | undefined, permission: PermissionKey): boolean`
  - `can(principal, required: PermissionKey | PermissionKey[] | undefined): boolean` — true when all required are held; `undefined` (no gate on a nav item) → true.
  - `type RoleFocus = "admin" | "supervisor" | "senior" | "technician" | "manager"`
  - `focusRole(principal): RoleFocus` — precedence: `workflow:publish` → admin; `incident:resolve` → supervisor; `execution:prerequisite:verify` → senior; `execution:write` → technician; `handover:accept` → manager; fallback → technician.

- [ ] **Step 1: Write the failing test**

Create `web/src/console/permissions.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { hasPermission, can, focusRole } from "./permissions";
import type { Principal } from "../types";

const principal = (permissions: string[]): Principal => ({
  id: "p1",
  display_name: "Tester",
  organization_id: "o1",
  site_ids: ["s1"],
  permissions,
});

describe("hasPermission", () => {
  it("is false without a principal", () => {
    expect(hasPermission(undefined, "report:write")).toBe(false);
  });

  it("is true when the permission is held", () => {
    expect(hasPermission(principal(["report:write"]), "report:write")).toBe(true);
  });
});

describe("can", () => {
  it("requires all listed permissions", () => {
    const p = principal(["report:write"]);
    expect(can(p, "report:write")).toBe(true);
    expect(can(p, ["report:write", "report:approve"])).toBe(false);
    expect(can(p, ["report:write"])).toBe(true);
  });
});

describe("focusRole", () => {
  it("maps each role's signature permission set to its focus", () => {
    const admin = ["workflow:publish", "incident:resolve", "execution:write", "report:approve"];
    const supervisor = ["incident:resolve", "report:approve", "execution:write"];
    const senior = ["execution:prerequisite:verify", "execution:write", "demonstration:review"];
    const technician = ["execution:write", "report:write"];
    const manager = ["handover:accept", "recommendation:review"];
    expect(focusRole(principal(admin))).toBe("admin");
    expect(focusRole(principal(supervisor))).toBe("supervisor");
    expect(focusRole(principal(senior))).toBe("senior");
    expect(focusRole(principal(technician))).toBe("technician");
    expect(focusRole(principal(manager))).toBe("manager");
  });

  it("falls back to technician for read-only principals", () => {
    expect(focusRole(principal(["incident:read"]))).toBe("technician");
    expect(focusRole(undefined)).toBe("technician");
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npx vitest run src/console/permissions.test.ts`
Expected: FAIL — module `./permissions` does not exist.

- [ ] **Step 3: Write the manifest**

Create `web/src/console/permissions.ts`:

```ts
import type { Principal } from "../types";

/** Every permission the console UI checks. Mirrors authorization.go. */
export type PermissionKey =
  | "asset:create"
  | "asset:criticality:approve"
  | "demonstration:capture"
  | "demonstration:review"
  | "execution:read:all"
  | "execution:write"
  | "execution:prerequisite:verify"
  | "handover:accept"
  | "handover:write"
  | "incident:create"
  | "incident:resolve"
  | "integration:external:import"
  | "knowledge:approve"
  | "knowledge:write"
  | "organization:create"
  | "recommendation:review"
  | "recommendation:run"
  | "report:approve"
  | "report:write"
  | "workflow:publish"
  | "workflow:review";

export function hasPermission(
  principal: Pick<Principal, "permissions"> | undefined,
  permission: PermissionKey,
): boolean {
  return principal?.permissions.includes(permission) ?? false;
}

export function can(
  principal: Pick<Principal, "permissions"> | undefined,
  required: PermissionKey | PermissionKey[] | undefined,
): boolean {
  if (!required) return true;
  const list = Array.isArray(required) ? required : [required];
  return list.every((permission) => hasPermission(principal, permission));
}

export type RoleFocus =
  | "admin"
  | "supervisor"
  | "senior"
  | "technician"
  | "manager";

/** Permission-derived focus, never role-name checks. Precedence matters. */
export function focusRole(
  principal: Pick<Principal, "permissions"> | undefined,
): RoleFocus {
  const has = (permission: PermissionKey) => hasPermission(principal, permission);
  if (has("workflow:publish")) return "admin";
  if (has("incident:resolve")) return "supervisor";
  if (has("execution:prerequisite:verify")) return "senior";
  if (has("execution:write")) return "technician";
  if (has("handover:accept")) return "manager";
  return "technician";
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `npx vitest run src/console/permissions.test.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add web/src/console/permissions.ts web/src/console/permissions.test.ts
git commit -m "feat: add web permission manifest with focus-role derivation"
```

---

### Task 3: PermissionGate component

**Files:**
- Create: `web/src/console/ui/PermissionGate.tsx`
- Test: `web/src/console/ui/PermissionGate.test.tsx`

**Interfaces:**
- Consumes: `can` + `PermissionKey` from `web/src/console/permissions.ts` (Task 2), `Principal` from `../../types`.
- Produces:
  - `PermissionGate({ principal, required, children })` — renders `children` only when `can(principal, required)`; otherwise null.

- [ ] **Step 1: Write the failing test**

Create `web/src/console/ui/PermissionGate.test.tsx`:

```tsx
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { PermissionGate } from "./PermissionGate";
import type { Principal } from "../../types";

const principal = (permissions: string[]): Principal => ({
  id: "p1",
  display_name: "Tester",
  organization_id: "o1",
  site_ids: ["s1"],
  permissions,
});

describe("PermissionGate", () => {
  it("renders children when the permission is held", () => {
    render(
      <PermissionGate principal={principal(["report:write"])} required="report:write">
        <button type="button">Draft report</button>
      </PermissionGate>,
    );
    expect(screen.getByRole("button", { name: "Draft report" })).toBeTruthy();
  });

  it("renders nothing when the permission is missing", () => {
    render(
      <PermissionGate principal={principal(["incident:read"])} required="report:write">
        <button type="button">Draft report</button>
      </PermissionGate>,
    );
    expect(screen.queryByRole("button", { name: "Draft report" })).toBeNull();
  });

  it("supports an AND list of permissions", () => {
    render(
      <PermissionGate
        principal={principal(["report:write", "report:approve"])}
        required={["report:write", "report:approve"]}
      >
        <span>Approve</span>
      </PermissionGate>,
    );
    expect(screen.getByText("Approve")).toBeTruthy();
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npx vitest run src/console/ui/PermissionGate.test.tsx`
Expected: FAIL — module `./PermissionGate` does not exist.

- [ ] **Step 3: Write the component**

Create `web/src/console/ui/PermissionGate.tsx`:

```tsx
import type { ReactNode } from "react";
import type { Principal } from "../../types";
import { can, type PermissionKey } from "../permissions";

/**
 * PermissionGate: renders children only when the principal holds every
 * required permission. Missing permission hides the subtree entirely.
 */
export function PermissionGate({
  principal,
  required,
  children,
}: {
  principal?: Pick<Principal, "permissions">;
  required: PermissionKey | PermissionKey[];
  children: ReactNode;
}) {
  return can(principal, required) ? <>{children}</> : null;
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `npx vitest run src/console/ui/PermissionGate.test.tsx`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add web/src/console/ui/PermissionGate.tsx web/src/console/ui/PermissionGate.test.tsx
git commit -m "feat: add PermissionGate for permission-gated UI"
```

---

### Task 4: Wire Sidebar nav gating to the manifest

**Files:**
- Modify: `web/src/console/layout/Sidebar.tsx:21-68`
- Test: `web/src/console/layout/Sidebar.test.tsx` (extend)

**Interfaces:**
- Consumes: `can`, `PermissionKey` from `../permissions` (Task 2).
- Produces: Sidebar nav-items gated through the shared manifest; `NavItem.permission: PermissionKey | undefined`.

- [ ] **Step 1: Write the regression tests**

This task is a refactor: behavior stays identical, but gating moves from the
inline `has` helper to the shared manifest. The two tests below lock the
behavior so the refactor cannot regress it. Extend `web/src/console/layout/Sidebar.test.tsx` with a parametrized principal helper and two new tests (keep the existing three tests unchanged):

```tsx
function renderSidebarWithPermissions(permissions: string[]) {
  return render(
    <ThemeProvider>
      <I18nProvider>
        <MemoryRouter initialEntries={["/"]}>
          <Routes>
            <Route path="*" element={<div>page</div>} />
          </Routes>
          <Sidebar
            principal={{
              id: "p1",
              display_name: "T",
              organization_id: "o1",
              site_ids: ["s1"],
              permissions,
            }}
          />
        </MemoryRouter>
      </I18nProvider>
    </ThemeProvider>,
  );
}
```

```tsx
  it("shows Reports for a technician (report:write)", () => {
    renderSidebarWithPermissions(["report:write", "execution:write"]);
    expect(screen.getByRole("link", { name: "Reports" })).toBeTruthy();
  });

  it("hides Reports for a manager (no report:write)", () => {
    renderSidebarWithPermissions(["handover:accept", "recommendation:review"]);
    expect(screen.queryByRole("link", { name: "Reports" })).toBeNull();
  });
```

Note: the Reports nav item already carries `permission: "report:write"`
(Sidebar.tsx:43) and the current inline `has` helper already hides it for the
manager, so both tests are expected to pass before the refactor.

- [ ] **Step 2: Run the tests to verify they pass (regression lock)**

Run: `npx vitest run src/console/layout/Sidebar.test.tsx`
Expected: PASS — all five tests (3 existing + 2 new) against the current
inline `has` helper. This is the baseline; Step 4 must keep them green.

- [ ] **Step 3: Wire the manifest into the Sidebar**

In `web/src/console/layout/Sidebar.tsx`:

Replace the `NavItem` type and the `has` helper (lines 21, 67-68):

```tsx
import { can, type PermissionKey } from "../permissions";

type NavItem = { to: string; key: MessageKey; icon: Icon; permission?: PermissionKey };
```

```tsx
  const visible = section.items.filter((item) => can(principal, item.permission));
```

Delete the old `has` helper function. The Reports item keeps `permission: "report:write"` (now typed as `PermissionKey`).

- [ ] **Step 4: Run the tests to verify the refactor stays green**

Run: `npx vitest run src/console/layout/Sidebar.test.tsx`
Expected: PASS — all five tests (3 existing + 2 new) against the manifest.
Also run `npm run lint` from `web/` to confirm the `PermissionKey` typing on
`NavItem.permission` and the `can(principal, item.permission)` call with
`permission: PermissionKey | undefined` type-check (Task 2's `can` accepts
`undefined`).

- [ ] **Step 5: Commit**

```bash
git add web/src/console/layout/Sidebar.tsx web/src/console/layout/Sidebar.test.tsx
git commit -m "refactor: gate sidebar nav through the shared permission manifest"
```

---

### Task 5: Role-aware dashboard focus panel

**Files:**
- Create: `web/src/console/dashboardFocus.ts`
- Modify: `web/src/console/pages/DashboardPage.tsx`
- Modify: `web/src/console/components/Overview.tsx`
- Modify: `web/src/console/pages/DashboardPage.test.tsx`
- Modify: `web/src/i18n/messages.ts` (en block near line 56, vi block near line 693)

**Interfaces:**
- Consumes: `RoleFocus` + `focusRole` from `./permissions` (Task 2); `MessageKey` from `../i18n/messages`.
- Produces:
  - `type FocusLink = { labelKey: MessageKey; to: string }`
  - `type FocusConfig = { titleKey: MessageKey; links: FocusLink[] }`
  - `FOCUS_CONFIG: Record<RoleFocus, FocusConfig>` — exported for tests.
  - `Overview` gains prop `focus: FocusConfig`; `DashboardPage` passes `FOCUS_CONFIG[focusRole(principal)]`.

- [ ] **Step 1: Add the i18n keys**

In `web/src/i18n/messages.ts`, add to the `en` block after `"dashboard.table.age"` (line 60):

```ts
  "dashboard.focus": "Focus",
  "dashboard.focus.admin": "Organization health",
  "dashboard.focus.supervisor": "Approvals & open incidents",
  "dashboard.focus.senior": "My executions & measurements due",
  "dashboard.focus.technician": "Assigned executions",
  "dashboard.focus.manager": "Handovers & recommendations",
```

Add to the `vi` block after `"dashboard.table.age"` (line 697):

```ts
  "dashboard.focus": "Trọng tâm",
  "dashboard.focus.admin": "Sức khỏe tổ chức",
  "dashboard.focus.supervisor": "Duyệt & sự cố đang mở",
  "dashboard.focus.senior": "Quy trình của tôi & phép đo đến hạn",
  "dashboard.focus.technician": "Công việc được giao",
  "dashboard.focus.manager": "Bàn giao & khuyến nghị",
```

- [ ] **Step 2: Write the failing tests**

Extend `web/src/console/pages/DashboardPage.test.tsx` — the mock at the top currently returns `permissions: ["incident:read"]` for `api.principal`; add three new tests that override the mock per case:

```tsx
  it("renders the admin focus panel for a publisher", async () => {
    vi.mocked(api.principal).mockResolvedValueOnce({
      id: "p1",
      display_name: "Admin",
      site_ids: [],
      permissions: ["workflow:publish", "incident:resolve", "report:approve"],
    });
    renderDashboard();
    expect(await screen.findByText("Organization health")).toBeTruthy();
  });

  it("renders the supervisor focus panel", async () => {
    vi.mocked(api.principal).mockResolvedValueOnce({
      id: "p1",
      display_name: "Supervisor",
      site_ids: [],
      permissions: ["incident:resolve", "report:approve"],
    });
    renderDashboard();
    expect(await screen.findByText("Approvals & open incidents")).toBeTruthy();
  });

  it("renders the manager focus panel", async () => {
    vi.mocked(api.principal).mockResolvedValueOnce({
      id: "p1",
      display_name: "Manager",
      site_ids: [],
      permissions: ["handover:accept", "recommendation:review"],
    });
    renderDashboard();
    expect(await screen.findByText("Handovers & recommendations")).toBeTruthy();
  });
```

Add a `renderDashboard()` helper in that file that wraps the existing
`render(...)` call (MemoryRouter + `I18nProvider` + `PrincipalProvider` +
`SiteProvider`, matching the current tests), and refactor the existing tests
to use it if they currently inline the render call. The existing first test
asserts the technician fallback focus implicitly — leave its assertions
untouched.

- [ ] **Step 3: Run the tests to verify they fail**

Run: `npx vitest run src/console/pages/DashboardPage.test.tsx`
Expected: the three new tests FAIL (`findByText` times out — focus panel not rendered).

- [ ] **Step 4: Write the focus config**

Create `web/src/console/dashboardFocus.ts`:

```ts
import type { MessageKey } from "../i18n/messages";
import type { RoleFocus } from "./permissions";

export type FocusLink = { labelKey: MessageKey; to: string };
export type FocusConfig = { titleKey: MessageKey; links: FocusLink[] };

/** Per-role dashboard focus, keyed by the permission-derived RoleFocus. */
export const FOCUS_CONFIG: Record<RoleFocus, FocusConfig> = {
  admin: {
    titleKey: "dashboard.focus.admin",
    links: [
      { labelKey: "nav.quality", to: "/quality" },
      { labelKey: "dashboard.openIncidents", to: "/incidents" },
      { labelKey: "dashboard.criticalAssets", to: "/assets" },
    ],
  },
  supervisor: {
    titleKey: "dashboard.focus.supervisor",
    links: [
      { labelKey: "dashboard.openIncidents", to: "/incidents" },
      { labelKey: "dashboard.pendingReports", to: "/reports" },
      { labelKey: "nav.quality", to: "/quality" },
    ],
  },
  senior: {
    titleKey: "dashboard.focus.senior",
    links: [
      { labelKey: "dashboard.assignedExecutions", to: "/executions" },
      { labelKey: "dashboard.pendingHandovers", to: "/handovers" },
      { labelKey: "nav.knowledge", to: "/knowledge" },
    ],
  },
  technician: {
    titleKey: "dashboard.focus.technician",
    links: [
      { labelKey: "dashboard.assignedExecutions", to: "/executions" },
      { labelKey: "nav.knowledge", to: "/knowledge" },
    ],
  },
  manager: {
    titleKey: "dashboard.focus.manager",
    links: [
      { labelKey: "dashboard.pendingHandovers", to: "/handovers" },
      { labelKey: "dashboard.openIncidents", to: "/incidents" },
      { labelKey: "nav.quality", to: "/quality" },
    ],
  },
};
```

- [ ] **Step 5: Render the focus panel in Overview**

In `web/src/console/components/Overview.tsx`:
- Add `focus` to the props type and destructure it.
- Import `type { FocusConfig } from "../dashboardFocus"`.
- Render the panel between the `metrics` section (line 81) and the "My queue" section (line 83):

```tsx
      <section className="panel" style={{ marginBottom: 20 }}>
        <div className="panel-heading">
          <h2>{t(focus.titleKey)}</h2>
        </div>
        <div className="focus-links">
          {focus.links.map((link) => (
            <Link key={link.to} to={link.to} className="secondary-button">
              {t(link.labelKey)}
            </Link>
          ))}
        </div>
      </section>
```

Add the `.focus-links` rule to `web/src/styles.css` (flex row, gap 8px, wrap; buttons are small `secondary-button`s):

```css
.focus-links {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
```

- [ ] **Step 6: Wire the focus into DashboardPage**

In `web/src/console/pages/DashboardPage.tsx`:

```tsx
import { focusRole } from "../permissions";
import { FOCUS_CONFIG } from "../dashboardFocus";
```

```tsx
  const focus = FOCUS_CONFIG[focusRole(principal)];
```

```tsx
        <Overview
          focus={focus}
          incidents={incidents.data ?? []}
          ...
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `npx vitest run src/console/pages/DashboardPage.test.tsx`
Expected: PASS — all tests including the three new focus-panel tests.

- [ ] **Step 8: Type-check and build**

Run: `npm run lint && npm run build` (from `web/`)
Expected: both green. `npm run lint` catches any missing vi key via `MessageKey` typing; `npm run build` runs `tsc -b` + Vite.

- [ ] **Step 9: Commit**

```bash
git add web/src/console/dashboardFocus.ts web/src/console/components/Overview.tsx \
  web/src/console/pages/DashboardPage.tsx web/src/console/pages/DashboardPage.test.tsx \
  web/src/i18n/messages.ts web/src/styles.css
git commit -m "feat: make dashboard focus role-aware per the role matrix"
```

---

## Final Verification (before declaring done)

Run from repo root, all must pass:

```bash
go test ./internal/identity/domain/
cd web && npx vitest run && npm run lint && npm run build
```

Then confirm `git status` shows only expected files and the branch is clean of unintended changes.

## Out of Scope (from the spec)

- Cluster B screens (users & roles, orgs & sites, settings, integrations, audit trail, recommendations review queue).
- Backend `/me` changes (focus is permission-derived; `roles` are not needed).
- Redesign of demonstrations / workflows / quality pages.

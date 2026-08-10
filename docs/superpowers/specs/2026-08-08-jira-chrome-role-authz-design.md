# Design: Jira-style chrome and per-role authorization screens

Date: 2026-08-08
Status: Approved (brainstorming session)

## Problem

The console already follows Jira's information architecture (grouped sidebar,
"For you" tabs, dense queues), but the chrome still reads as an instrument
panel rather than Jira's app frame: the brand lives in the sidebar, there is
no global Create action, and no account menu. Role differences are expressed
only through hidden buttons; there is no explicit authorization surface, so a
restricted user gets silent omissions instead of a clear "you are role X,
this area needs permission Y" screen.

## Goals

1. Jira-like top bar: brand at left, centered search, primary Create action,
   avatar menu at right.
2. Explicit authorization screens per role: restricted routes render a
   "Not authorized" screen naming the user's roles and the missing
   permission; sidebar hides gated links consistently.
3. A per-role account screen (avatar menu, Profile) showing identity, roles,
   and the permission set, plus a demo role preview switcher for the five
   pilot roles.

## Non-goals

- Cluster B admin screens (users & roles, settings, integrations, audit).
- Backend role switching; preview is frontend navigation-only. The API
  continues to enforce real permissions.
- Atlassian blue or Jira visual branding; Skawld emerald tokens stay.

## Components

### Top bar (`GlobalBar` restyle)

- Left: existing `.brand` block moved from `Sidebar` into the top bar.
- Center: search field (existing CommandPalette trigger, restyled as an
  input-like button with `/` kbd hint).
- Right: site switcher, theme toggle, language select (unchanged behavior),
  then a primary emerald `+ Create` button with dropdown menu:
  - "New incident" (needs `incident:create`) navigates `/incidents?create=1`
  - "New asset" (needs `asset:create`) navigates `/assets?create=1`
  Items the principal cannot perform are hidden.
- Avatar button (initials circle from `display_name`) opens the account menu:
  name, roles line, "Profile" -> `/account`, "Theme" toggles theme,
  "Sign out" submits the existing POST `/auth/logout` form.

### Backend: `/me` exposes roles

`currentPrincipal` in `internal/platform/httpserver/router.go` adds
`"roles": principal.Roles` (strings). Web `Principal` type gains
`roles: string[]`.

### Route authorization

- `web/src/console/rolePermissions.ts`: mirror of
  `PermissionsForRole` from `internal/identity/domain/authorization.go` for
  the five roles (Administrator, Maintenance Supervisor, Senior Technician,
  Technician, Manager). Used only for the account screen display and the
  preview persona; never for enforcement.
- `web/src/console/state/PreviewProvider.tsx`: sessionStorage-backed preview
  role (`skawld.previewRole`). When set, `usePrincipal` overlays the preview
  role's permission mirror over the real principal's permissions and exposes
  `previewRole` + `exitPreview`. A dismissible banner in `ConsoleLayout`
  shows "Previewing as X" with an exit action.
- `AuthorizedRoute` wrapper in `App.tsx`:
  - `/quality` requires `recommendation:review`
  - `/reports` and `/reports/:id` require `report:write` or `report:approve`
  Missing permission renders `NotAuthorizedPage` instead of the page.
- `NotAuthorizedPage`: lock icon, "You do not have access", the principal's
  roles, the missing permission in mono, links to `/` and `/account`.
- Sidebar: Reports link gated on `report:write` or `report:approve`
  (currently `report:write` only, which wrongly hides it from managers who
  hold `report:approve`? No: managers do not hold report:write; the backend
  list requires write OR approve, so the nav gate must match), Quality link
  gated on `recommendation:review`.

### Account page (`/account`)

- Identity card: avatar initials, display name, organization id, site ids.
- Role badges from `principal.roles`.
- Permission chips grouped by area (assets, incidents, executions,
  knowledge, reports, handovers, demonstrations, workflows, governance),
  rendered from the real `principal.permissions`.
- "Preview as role" section: five role buttons using the
  `rolePermissions` mirror; selecting one sets the preview persona and
  navigates home; current preview highlighted; "Use my real role" exits.

## i18n

All new copy added to `web/src/i18n/messages.ts` in en and vi under keys
prefixed `topbar.`, `menu.`, `authz.`, `account.`.

## Verification

- `npx tsc -b` clean; `npm test` green (new tests: GlobalBar menu, avatar
  menu, AuthorizedRoute, NotAuthorizedPage, AccountPage, preview overlay).
- `web/scripts/verify-console.py` selectors unchanged: sidebar shows
  "Reports" for supervisor (holds `report:write`) and hides it for manager
  (holds neither `report:write` nor `report:approve`); the script's
  expectations hold.

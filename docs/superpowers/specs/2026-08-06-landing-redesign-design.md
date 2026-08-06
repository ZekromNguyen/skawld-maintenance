# Skawld Landing Redesign — Design

Date: 2026-08-06
Status: Approved direction (scope C, signature = drafting grid + procedure meter, hamburger mobile nav, Playwright overflow test)

## 1. Problem

The marketing landing page (`web/src/marketing`, route `/landing`) has six
reproducible defects and reads as a generic dark-SaaS template rather than a
distinctive product page. This spec fixes the defects and elevates the design
within the locked brand system (docs/contributing/web-ui-design-system.md).

### 1.1 Defect list (all verified in code)

1. **Mobile header overflow (390px)** — `SiteHeader.tsx:56-84`: the right-side
   cluster (locale, Sign in, Book a pilot) never collapses below 1024px; the
   CTA reaches past the 390px viewport. `.nav-mobile` (marketing.css:117)
   renders as a redundant second row at every size under 1024px.
2. **Duplicate "P-302 · P-302"** — `Hero.tsx:77` hardcodes `P-302 · ` while
   i18n `landing.hero.panelTitle` already starts with "P-302 ·".
3. **Slack icon fails to load** — `Integrations.tsx:53` uses
   `cdn.simpleicons.org`; CDN-dependent, `naturalWidth` is 0.
4. **404 console errors** — (a) `/openapi.yaml`: linked from PilotCta.tsx and
   SiteFooter.tsx but only exists at `api/openapi.yaml`; vite proxies only
   `/api`, `/auth`, `/health`. (b) No `web/public/` directory exists, so there
   is no favicon (`/favicon.ico` 404).
5. **Inverted/weak type hierarchy** — `FeaturesBento.tsx:59`: card h3 at 16.5px
   vs body 13.5px (+3px), inconsistent with h3s elsewhere (WorkflowRail 19px,
   UseCases 24px).
6. **No mobile overflow regression test** — vitest + jsdom cannot measure
   `scrollWidth` (no layout engine).

## 2. Design direction

Design read (unchanged, from the design system doc): "B2B enterprise
industrial SaaS, dark-tech precision language, instrument-emerald identity,
premium comparable to Linear." The brand palette, type, shape, and motion rules
are locked and are the brief. Freedom is spent on layout and one signature.

**Concept: "the work order, drawn to scale."** The page is an engineering
artifact: hairline drafting grid, mono instrument readouts, and the product's
real workflow semantics (capture/review/publish/assist/verify with LOTO gates)
as the recurring visual grammar. One CTA intent ("Book a pilot") throughout.

### 2.1 Signature (the one memorable element)

The **procedure meter + drafting grid**, used together but quietly:
- A faint hairline Cartesian grid (1px lines, `--brand` at ~7%, masked
  radial) behind the hero and the conversion band.
- A thin procedure meter: the five phase labels on a 1px rail with tick
  marks and the current phase lit (semantic state: "EXECUTION IN PROGRESS"),
  spanning the hero and echoed by the bento peek cell and workflow rail.
- This is true structure (the product's own step states), not decoration; it
  does not use numbered section eyebrows (banned) — numbering only appears in
  the workflow rail, which is a real sequence.

### 2.2 Tokens (additive only, no new accents)

| Token | Value | Use |
|---|---|---|
| `--grid-line` | `color-mix(in srgb, var(--brand) 7%, transparent)` | Drafting grid hairlines |
| `--callout-line` | `var(--line-soft)` | Leader lines / callout rules |

Palette, dual-mode, and all existing tokens unchanged.

### 2.3 Type scale (normalized, monotonic)

| Role | Size / Weight / Leading / Tracking |
|---|---|
| Hero H1 | `clamp(38px, 5.5vw, 64px)` / 700 / 1.0 / `-0.03em` |
| Section title | `clamp(30px, 4vw, 46px)` / 700 / 1.05 / `-0.025em` |
| Bento card title | 18px / 700 / 1.3 / 0 |
| Workflow step title | 20px / 650 / 1.3 / 0 |
| Tab panel title | 24px / 700 / 1.2 / 0 |
| Body / section lead | 15px / 1.65 / `--ink-soft` |
| Card body | 14px / 1.6 / `--ink-soft` |
| Mono label | 11px / 600 / uppercase / `0.16em` letter-spacing |
| Mono value | Geist Mono / 700 |

## 3. Component changes

### 3.1 SiteHeader (fixes #1)

- ≥1024px: unchanged desktop row (logo, `nav-desktop`, locale, Sign in,
  Book a pilot).
- <1024px: logo left; right side = compact "Book a pilot" + hamburger button
  (Phosphor `List`, aria-expanded). No `nav-mobile` row (delete it and its CSS).
- Menu: Radix Dialog (already a dependency) containing nav links, locale
  select, and Sign in. Focus trap, ESC, and label wiring from Radix.
- New i18n keys (EN + VI): `landing.nav.menu`, `landing.nav.close`.

### 3.2 Hero (fixes #2; signature)

- Remove the hardcoded `P-302 · ` prefix at Hero.tsx:77; render
  `t("landing.hero.panelTitle")` plus a new mono "WORK ORDER" label
  (`landing.hero.panelLabel`) in the panel header.
- Panel: instrument-face rework — drafting-grid backdrop behind the section,
  header row = mono label + panel title + status chip (unchanged status text).
  Step rail and metrics row keep their real data; metrics row gains one
  leader-line callout (`PUMP P-302 · 8.1 mm/s RMS` style, mono, `--callout-line`).
- Procedure meter: 1px rail under the copy column with five phase ticks,
  current phase lit, matching the step rail states.
- Hero H1 up to `clamp(38, 5.5vw, 64)`; subtitle stays ≤ 20 words.

### 3.3 FeaturesBento (fixes #5)

- Card title: 18px / 700; body 14px / 1.6; cell gap 14px. Icon tile 40px stays.
- Tinted cells and WorkflowPeekCell stay (bento diversity rule); peek cell
  dots restyle to match the procedure meter grammar (same dot states).

### 3.4 WorkflowRail

- Keep the numbered rail (real sequence) and gate chips.
- Step titles to 20px / 650; body 14.5px / 1.6.

### 3.5 Integrations (fixes #3)

- Replace the Simple Icons CDN `<img>` with self-contained monogram tiles:
  28px rounded square, mono 800 uppercase initial, `--surface` bg,
  `--brand-dim` border, optional curated per-brand hue (GitHub `#f0f6fc`,
  Slack `#611f69`, PostgreSQL `#336791`, Docker `#2496ED`, MinIO `#c72e49`,
  Keycloak `#008aaa`) applied as a faint border/fill tint; tooltip =
  name. Same pattern as LogoWall. No network dependency, no hand-rolled
  multi-path SVGs (monogram letters only, per the icon rule).

### 3.6 PilotCta

- Drafting-grid backdrop behind the band; keep gradient tint.
- Keep single CTA intent + "Read the API spec" ghost button (now resolves, see
  3.8). New copy optional; no em-dashes.

### 3.7 SiteFooter

- Keep structure; `landing.footer.openapi` link now resolves (3.8).
- Column headings stay 11px mono uppercase.

### 3.8 Assets (fixes #4)

- `web/public/favicon.svg`: the Logo "S" tile motif (rounded square, emerald
  border, mono S) as a self-contained SVG; reference from `web/index.html`
  via `<link rel="icon" href="/favicon.svg">`.
- `web/public/openapi.yaml`: copy of `api/openapi.yaml` so the web app serves
  it; add a build/refresh step (Makefile or script) to keep it in sync.
  Drift risk documented in the copy step.

## 4. Copy rules (unchanged, audited)

- Zero em-dashes anywhere visible; one CTA label ("Book a pilot"); sentence
  case; active voice; no filler verbs. All copy through i18n (EN + VI).
- New keys: `landing.nav.menu`, `landing.nav.close`, `landing.hero.panelLabel`
  (+ VI equivalents).

## 5. Accessibility

- Skip link stays; one h1; ordered headings (h1 → h2 sections → h3 cards).
- Radix Dialog gives focus trap, ESC close, and `aria-label` via new keys.
- Visible `:focus-visible` rings on all new controls (existing rule).
- `prefers-reduced-motion`: existing collapse rules cover Reveal; new elements
  (grid, meter) are static, nothing new to animate.

## 6. Testing

### 6.1 Unit (vitest + jsdom, `web/src/marketing/Landing.test.tsx`)

- Update header assertions if markup changes (menu button present; nav links
  exist inside the dialog; "Sign in" present at desktop breakpoint mock).
- New structural mobile test: hamburger button renders, `aria-expanded`
  toggles, nav links reachable from the menu.
- Existing tests (sections present, FAQ, EN/VI toggle, VI FAQ) must stay green.

### 6.2 E2E (new: @playwright/test, fixes #6)

- Add `@playwright/test` to web devDependencies; `playwright.config.ts` with
  a `webServer` running `vite dev` (port 5173).
- `e2e/mobile-overflow.spec.ts`: viewport 390x844; assert
  `document.documentElement.scrollWidth <= clientWidth`; open the hamburger
  menu; assert no horizontal scroll while open; assert CTA visible.
- Second spec: console-error check that `/openapi.yaml` and `/favicon.svg`
  return 200 (regression for #4).

## 7. Acceptance criteria

1. At 390px: no horizontal overflow; header fits; menu opens/closes with
   keyboard; nav links reachable. (#1, #6)
2. Panel header shows exactly one "P-302". (#2)
3. Integrations render without network requests; Slack tile visible. (#3)
4. No 404 in console: `/favicon.svg` and `/openapi.yaml` return 200. (#4)
5. Bento h3 ≥ 18px, clearly above body 14px; consistent across sections. (#5)
6. All existing vitest tests pass; new vitest + Playwright tests pass. (#6)
7. Dual-mode (dark/light) and EN/VI both render correctly.

## 8. Out of scope

- Console (root `/`) UI.
- Backend changes (openapi.yaml stays the source at `api/openapi.yaml`).
- New marketing sections or content.

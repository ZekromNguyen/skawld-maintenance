# Skawld Landing Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the six landing-page defects and elevate the design with a drafting-grid + procedure-meter signature, all within the locked brand system.

**Architecture:** Pure frontend work in `web/`. All copy moves through the existing i18n provider (additive EN/VI keys only). Layout primitives keep using `marketing.css` classes; content cards keep the existing inline-style + CSS-variable pattern. The mobile header swaps the redundant `nav-mobile` row for a Radix Dialog hamburger menu. A new Playwright e2e suite covers the 390px overflow regression and the static-asset 404s that jsdom cannot detect.

**Tech Stack:** React 19, Vite 8, Tailwind v4 (unused on marketing; plain CSS + inline styles), vitest + jsdom + Testing Library, @radix-ui/react-dialog (already a dependency), @phosphor-icons/react, @playwright/test (new dev dependency).

## Global Constraints

- **Palette:** locked tokens from `web/src/marketing/marketing.css`. Only additive tokens allowed: `--grid-line` and `--callout-line`. No new accent colors.
- **Type:** Geist + Geist Mono only. Mono labels 11px/0.16em uppercase. Card titles ≥ 18px (fixes inverted hierarchy).
- **Copy:** exactly one CTA intent label, "Book a pilot" / "Đặt lịch thí điểm". Zero em-dashes anywhere visible. No filler verbs.
- **i18n:** every user-visible string goes through `t()`. `MessageKey = keyof typeof en` and `const vi: Record<MessageKey, string>` mean a new EN key is a compile error without its VI equivalent — always add both.
- **Shape lock:** pill buttons (radius 999px), cards 16px, inputs 10px.
- **Motion:** animate `transform`/`opacity` only; honor `prefers-reduced-motion` (existing `marketing.css` collapse rules cover new static elements automatically).
- **Icons:** Phosphor only. Monogram tiles are text letters (like `LogoWall.tsx`), not hand-rolled SVG paths.
- **Patterns:** follow existing files — class-based layout primitives, inline styles for card internals. Do not restructure `messages.ts`.

---

### Task 1: Add i18n keys for menu, panel label, and callout

**Files:**
- Modify: `web/src/i18n/messages.ts:343` (EN, after `landing.nav.faq`), `:352` (EN, after `landing.hero.panelTitle`), `:832` (VI, after `landing.nav.faq`), `:841` (VI, after `landing.hero.panelTitle`)
- Test: `web/src/i18n/messages.test.ts` (create)

**Interfaces:**
- Produces: message keys `landing.nav.menu`, `landing.nav.close`, `landing.hero.panelLabel`, `landing.hero.callout` in both locales. Tasks 2 and 4 consume these exact keys.

- [ ] **Step 1: Write the failing test**

Create `web/src/i18n/messages.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { messages } from "./messages";

describe("landing i18n keys", () => {
  it("provides the new menu and hero panel keys in both locales", () => {
    for (const locale of ["en", "vi"] as const) {
      expect(messages[locale]["landing.nav.menu"].length).toBeGreaterThan(0);
      expect(messages[locale]["landing.nav.close"].length).toBeGreaterThan(0);
      expect(messages[locale]["landing.hero.panelLabel"].length).toBeGreaterThan(0);
      expect(messages[locale]["landing.hero.callout"].length).toBeGreaterThan(0);
    }
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/i18n/messages.test.ts`
Expected: FAIL — `messages[locale]["landing.nav.menu"]` is `undefined`.

- [ ] **Step 3: Add the EN keys**

In `web/src/i18n/messages.ts`, replace:

```ts
  "landing.nav.faq": "FAQ",
```

with:

```ts
  "landing.nav.faq": "FAQ",
  "landing.nav.menu": "Menu",
  "landing.nav.close": "Close menu",
```

and replace:

```ts
  "landing.hero.panelTitle": "P-302 · Circulation pump inspection",
```

with:

```ts
  "landing.hero.panelTitle": "P-302 · Circulation pump inspection",
  "landing.hero.panelLabel": "Work order",
  "landing.hero.callout": "P-302 · shaft vibration at 8.1 mm/s RMS",
```

- [ ] **Step 4: Add the VI keys**

Replace:

```ts
  "landing.nav.faq": "Câu hỏi thường gặp",
```

with:

```ts
  "landing.nav.faq": "Câu hỏi thường gặp",
  "landing.nav.menu": "Trình đơn",
  "landing.nav.close": "Đóng trình đơn",
```

and replace:

```ts
  "landing.hero.panelTitle": "P-302 · Kiểm tra bơm tuần hoàn",
```

with:

```ts
  "landing.hero.panelTitle": "P-302 · Kiểm tra bơm tuần hoàn",
  "landing.hero.panelLabel": "Lệnh công việc",
  "landing.hero.callout": "P-302 · rung động trục ở 8,1 mm/s RMS",
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd web && npx vitest run src/i18n/messages.test.ts`
Expected: PASS.

- [ ] **Step 6: Type-check and commit**

Run: `cd web && npm run lint`
Expected: no errors.

```bash
git add web/src/i18n/messages.ts web/src/i18n/messages.test.ts
git commit -m "feat(web): add i18n keys for header menu and hero callout"
```

---

### Task 2: Hero — de-duplicate P-302, panel label, grid backdrop, procedure meter, callout

**Files:**
- Modify: `web/src/marketing/components/Hero.tsx`
- Modify: `web/src/marketing/marketing.css:6-44` (add `--grid-line`, `--callout-line` tokens)
- Test: `web/src/marketing/Landing.test.tsx` (extend)

**Interfaces:**
- Consumes: `landing.hero.panelLabel`, `landing.hero.callout`, `landing.phase.*` (all exist after Task 1).
- Produces: `data-current="true"` attribute on the procedure meter's current-phase dot (Task 7's e2e and this task's unit test rely on it). Element class `hero-meter` (query target).

- [ ] **Step 1: Add the two additive tokens**

In `web/src/marketing/marketing.css`, inside `:root { ... }` after the `--blue` line, add:

```css
  --grid-line: color-mix(in srgb, var(--brand) 7%, transparent);
  --callout-line: var(--line-soft);
```

- [ ] **Step 2: Write the failing tests**

Append to `web/src/marketing/Landing.test.tsx` inside the existing `describe` block:

```ts
  it("shows exactly one P-302 tag in the hero panel header", () => {
    renderLanding();
    expect(document.body.textContent).not.toContain("P-302 · P-302");
    expect(screen.getByText("Work order")).toBeTruthy();
    expect(screen.getByText("P-302 · Circulation pump inspection")).toBeTruthy();
  });

  it("renders the procedure meter with exactly one current phase", () => {
    renderLanding();
    const current = document.querySelectorAll('.hero-meter [data-current="true"]');
    expect(current.length).toBe(1);
    expect((current[0] as HTMLElement).style.background).toBe("var(--amber)");
  });
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd web && npx vitest run src/marketing/Landing.test.tsx`
Expected: FAIL — "P-302 · P-302" is present; no "Work order" text; no `.hero-meter` element.

- [ ] **Step 4: Fix the duplicated tag and add the WORK ORDER label**

In `web/src/marketing/components/Hero.tsx`, replace the panel-header title span (currently `P-302 · {t("landing.hero.panelTitle")}` inside the header row) with a two-line label block:

```tsx
        <span style={{ display: "grid", gap: 2 }}>
          <span
            style={{
              fontSize: 10,
              fontFamily: '"Geist Mono Variable", monospace',
              color: "var(--brand)",
              textTransform: "uppercase",
              letterSpacing: "0.14em",
            }}
          >
            {t("landing.hero.panelLabel")}
          </span>
          <span
            style={{
              fontSize: 12,
              fontWeight: 600,
              color: "var(--ink)",
              letterSpacing: "0.01em",
            }}
          >
            {t("landing.hero.panelTitle")}
          </span>
        </span>
```

- [ ] **Step 5: Add the metrics callout**

In the same file, after the metrics grid `</div>` (the `repeat(3, 1fr)` grid) and before the closing of the `padding: 18` wrapper div, add:

```tsx
        <div
          style={{
            fontSize: 9.5,
            fontFamily: '"Geist Mono Variable", monospace',
            color: "var(--ink-muted)",
            borderLeft: "1px solid var(--callout-line)",
            paddingLeft: 8,
            lineHeight: 1.5,
          }}
        >
          {t("landing.hero.callout")}
        </div>
```

- [ ] **Step 6: Add the grid backdrop to the hero section**

Add `import type { MessageKey } from "../../i18n/messages";` at the top of `Hero.tsx`. In the exported `Hero` component, change the `<section ...>` opening to include `position: "relative"`, then insert the backdrop as the first child:

```tsx
    <section
      style={{
        position: "relative",
        paddingTop: "clamp(64px, 8vw, 96px)",
        paddingBottom: "clamp(48px, 6vw, 72px)",
      }}
    >
      <div
        aria-hidden="true"
        style={{
          position: "absolute",
          inset: 0,
          pointerEvents: "none",
          backgroundImage:
            "linear-gradient(var(--grid-line) 1px, transparent 1px), linear-gradient(90deg, var(--grid-line) 1px, transparent 1px)",
          backgroundSize: "56px 56px",
          maskImage:
            "radial-gradient(ellipse 70% 55% at 50% 0%, black 25%, transparent 72%)",
          WebkitMaskImage:
            "radial-gradient(ellipse 70% 55% at 50% 0%, black 25%, transparent 72%)",
        }}
      />
      <div className="container hero-grid" style={{ position: "relative" }}>
```

- [ ] **Step 7: Add the procedure meter**

At module level in `Hero.tsx` (below `InstrumentPanel`, above the exported `Hero`), add:

```tsx
const PHASES: Array<{ labelKey: MessageKey; state: "done" | "current" | "pending" }> = [
  { labelKey: "landing.phase.capture", state: "done" },
  { labelKey: "landing.phase.review", state: "done" },
  { labelKey: "landing.phase.publish", state: "current" },
  { labelKey: "landing.phase.assist", state: "pending" },
  { labelKey: "landing.phase.verify", state: "pending" },
];

/** ProcedureMeter: the page's signature — the product's real workflow states
 * on a single rail, with the executing phase lit (same grammar as the bento
 * peek cell and the workflow rail). */
function ProcedureMeter() {
  const { t } = useI18n();
  return (
    <div
      className="hero-meter"
      role="group"
      aria-label={t("landing.hero.panelAria")}
      style={{ marginTop: 40, display: "flex", alignItems: "flex-start", gap: 0 }}
    >
      {PHASES.map((phase, index) => {
        const isCurrent = phase.state === "current";
        return (
          <div
            key={phase.labelKey}
            style={{ display: "flex", alignItems: "center", gap: 0, flex: 1, minWidth: 0 }}
          >
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 6 }}>
              <span
                data-current={isCurrent}
                style={{
                  width: 12,
                  height: 12,
                  borderRadius: "50%",
                  background: isCurrent
                    ? "var(--amber)"
                    : phase.state === "done"
                      ? "var(--brand)"
                      : "var(--line-soft)",
                  flexShrink: 0,
                }}
              />
              <span
                style={{
                  fontSize: 9.5,
                  fontFamily: '"Geist Mono Variable", monospace',
                  color: isCurrent ? "var(--ink)" : phase.state === "done" ? "var(--ink-soft)" : "var(--ink-muted)",
                  textTransform: "uppercase",
                  letterSpacing: "0.08em",
                  whiteSpace: "nowrap",
                }}
              >
                {t(phase.labelKey)}
              </span>
            </div>
            {index < PHASES.length - 1 ? (
              <span
                aria-hidden="true"
                style={{ flex: 1, height: 1, background: "var(--line)", margin: "0 10px", marginTop: 6 }}
              />
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
```

Then, in the left copy column of `Hero`, after the CTA row `</div>` (the flex-wrap row with the two buttons), add:

```tsx
            <ProcedureMeter />
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `cd web && npx vitest run src/marketing/Landing.test.tsx`
Expected: PASS (all existing + 2 new).

- [ ] **Step 9: Type-check and commit**

Run: `cd web && npm run lint`
Expected: no errors.

```bash
git add web/src/marketing/components/Hero.tsx web/src/marketing/marketing.css web/src/marketing/Landing.test.tsx
git commit -m "fix(web): de-duplicate hero tag and add procedure meter signature"
```

---

### Task 3: Integrations — self-contained monogram tiles

**Files:**
- Modify: `web/src/marketing/components/Integrations.tsx`
- Test: `web/src/marketing/Landing.test.tsx` (extend)

**Interfaces:**
- Produces: no external dependencies; each card is an `<a>` with a 28px monogram `<span aria-hidden="true">` plus the name text. Task 7's e2e relies on zero network requests for this section.

- [ ] **Step 1: Write the failing test**

Append to `web/src/marketing/Landing.test.tsx`:

```ts
  it("renders integrations as self-contained monogram tiles", () => {
    renderLanding();
    const integrations = document.getElementById("integrations");
    expect(integrations).toBeTruthy();
    expect(integrations!.querySelectorAll("img").length).toBe(0);
    expect(integrations!.querySelectorAll("a").length).toBe(6);
    expect(integrations!.textContent).toContain("Slack");
  });
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/marketing/Landing.test.tsx`
Expected: FAIL — the section still contains CDN `<img>` elements (and the "Slack" `<img>` has no text).

- [ ] **Step 3: Replace CDN images with monogram tiles**

Rewrite `web/src/marketing/components/Integrations.tsx` with:

```tsx
import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";
import { useI18n } from "../../i18n/I18nProvider";

/**
 * Integrations: compact card grid. Self-contained monogram tiles (text
 * letters, no hand-rolled SVG paths) instead of CDN icons, so nothing
 * depends on a third-party host at render time.
 */
const INTEGRATIONS: Array<{ name: string; href: string; initial: string; color: string }> = [
  { name: "Keycloak", href: "https://www.keycloak.org", initial: "K", color: "#008aaa" },
  { name: "PostgreSQL", href: "https://www.postgresql.org", initial: "P", color: "#336791" },
  { name: "S3 / MinIO", href: "https://min.io", initial: "M", color: "#c72e49" },
  { name: "GitHub", href: "https://github.com", initial: "G", color: "#f0f6fc" },
  { name: "Slack", href: "https://slack.com", initial: "S", color: "#611f69" },
  { name: "Docker", href: "https://www.docker.com", initial: "D", color: "#2496ED" },
];

function IntegrationCard({
  name,
  href,
  initial,
  color,
}: {
  name: string;
  href: string;
  initial: string;
  color: string;
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      title={name}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        border: "1px solid var(--line)",
        borderRadius: 16,
        padding: "18px 20px",
        background: "var(--surface)",
        textDecoration: "none",
        transition: "border-color 0.15s ease, background-color 0.15s ease",
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.borderColor = "var(--brand-dim)";
        e.currentTarget.style.background = "var(--raised)";
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.borderColor = "var(--line)";
        e.currentTarget.style.background = "var(--surface)";
      }}
    >
      <span
        aria-hidden="true"
        style={{
          width: 28,
          height: 28,
          display: "grid",
          placeItems: "center",
          borderRadius: 8,
          border: "1px solid",
          borderColor: `color-mix(in srgb, ${color} 45%, transparent)`,
          background: `color-mix(in srgb, ${color} 10%, var(--surface))`,
          fontFamily: '"Geist Mono Variable", monospace',
          fontSize: 12,
          fontWeight: 800,
          color: "var(--ink)",
          flexShrink: 0,
        }}
      >
        {initial}
      </span>
      <span style={{ fontSize: 14, fontWeight: 600, color: "var(--ink)" }}>
        {name}
      </span>
    </a>
  );
}

export function Integrations() {
  const { t } = useI18n();
  return (
    <Section
      id="integrations"
      title={t("landing.integrations.title")}
      lead={t("landing.integrations.lead")}
      className=""
    >
      <Reveal>
        <div className="card-grid">
          {INTEGRATIONS.map((integration) => (
            <IntegrationCard key={integration.name} {...integration} />
          ))}
        </div>
      </Reveal>
    </Section>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web && npx vitest run src/marketing/Landing.test.tsx`
Expected: PASS.

- [ ] **Step 5: Type-check and commit**

Run: `cd web && npm run lint`
Expected: no errors.

```bash
git add web/src/marketing/components/Integrations.tsx web/src/marketing/Landing.test.tsx
git commit -m "fix(web): replace CDN integration icons with self-contained monograms"
```

---

### Task 4: SiteHeader — responsive hamburger menu, remove nav-mobile row

**Files:**
- Modify: `web/src/marketing/components/SiteHeader.tsx` (rewrite)
- Modify: `web/src/marketing/marketing.css:91-134` (remove `.nav-mobile` rules, add header cluster classes, `.btn-compact`, `.visually-hidden`)
- Create: `web/src/test/setup.ts` (jsdom stubs for Radix Dialog)
- Modify: `web/vite.config.ts` (add `test.setupFiles`)
- Test: `web/src/marketing/Landing.test.tsx` (extend)

**Interfaces:**
- Consumes: `landing.nav.menu`, `landing.nav.close` (Task 1).
- Produces: classes `.header-actions-desktop` / `.header-actions-mobile` and `.btn-compact`; single locale `<select>` (always visible, so existing language tests keep passing); hamburger button with `aria-label` "Menu"; Radix `Dialog.Content` with `role="dialog"`.

- [ ] **Step 1: Add jsdom stubs and wire the setup file**

Create `web/src/test/setup.ts`:

```ts
// jsdom lacks a few browser APIs that @radix-ui/react-dialog touches.
// Stub them so the mobile menu renders and opens in unit tests.
if (typeof window !== "undefined") {
  if (!window.matchMedia) {
    Object.defineProperty(window, "matchMedia", {
      writable: true,
      value: (query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
      }),
    });
  }
  if (!window.ResizeObserver) {
    Object.defineProperty(window, "ResizeObserver", {
      writable: true,
      value: class ResizeObserver {
        observe() {}
        unobserve() {}
        disconnect() {}
      },
    });
  }
  if (!Element.prototype.scrollIntoView) {
    Element.prototype.scrollIntoView = () => {};
  }
}
```

In `web/vite.config.ts`, inside the `test` object, add:

```ts
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: "./src/test/setup.ts"
  }
```

- [ ] **Step 2: Write the failing test**

Append to `web/src/marketing/Landing.test.tsx`:

```ts
  it("opens the mobile menu and exposes nav links and sign-in", () => {
    renderLanding();
    const menuButton = screen.getByRole("button", { name: "Menu" });
    fireEvent.click(menuButton);
    const dialog = screen.getByRole("dialog");
    expect(dialog.textContent).toContain("Features");
    expect(dialog.textContent).toContain("Sign in");
  });

  it("keeps exactly one language switcher on the page", () => {
    renderLanding();
    expect(screen.getAllByLabelText(/language/i).length).toBe(1);
  });
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd web && npx vitest run src/marketing/Landing.test.tsx`
Expected: FAIL — no button named "Menu" (and the duplicate-select check currently fails because the select appears twice after the rewrite is skipped).

- [ ] **Step 4: Add the header CSS**

In `web/src/marketing/marketing.css`, replace the block:

```css
.landing .nav-desktop a,
.landing .nav-mobile a {
  color: var(--ink-soft);
  font-size: 14px;
  text-decoration: none;
  font-weight: 500;
  transition: color 0.15s ease;
}
```

with:

```css
.landing .nav-desktop a {
  color: var(--ink-soft);
  font-size: 14px;
  text-decoration: none;
  font-weight: 500;
  transition: color 0.15s ease;
}
```

Delete the whole `.nav-mobile` rule and its media query:

```css
.landing .nav-mobile {
  display: flex;
  gap: 18px;
  overflow-x: auto;
  padding-bottom: 12px;
  white-space: nowrap;
  scrollbar-width: none;
}

.landing .nav-mobile::-webkit-scrollbar {
  display: none;
}

@media (min-width: 1024px) {
  .landing .nav-mobile {
    display: none;
  }
}
```

Add the header cluster utilities and compact button after the `.btn-ghost:hover` rule:

```css
.landing .header-actions-desktop {
  display: none;
  align-items: center;
  gap: 10px;
}

.landing .header-actions-mobile {
  display: none;
  align-items: center;
  gap: 8px;
}

@media (max-width: 1023.98px) {
  .landing .header-actions-mobile {
    display: flex;
  }
}

@media (min-width: 1024px) {
  .landing .header-actions-desktop {
    display: flex;
  }
}

.landing .btn-compact {
  padding: 9px 14px;
  font-size: 13px;
}

.landing .visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
  border: 0;
}
```

- [ ] **Step 5: Rewrite SiteHeader.tsx**

Replace the whole file with:

```tsx
import { useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { List } from "@phosphor-icons/react";
import { Logo } from "./shared/Logo";
import { Button } from "./shared/Button";
import { useI18n } from "../../i18n/I18nProvider";
import type { Locale, MessageKey } from "../../i18n/messages";

const NAV_LINKS: Array<{ href: string; key: MessageKey }> = [
  { href: "#features", key: "landing.nav.features" },
  { href: "#workflow", key: "landing.nav.howItWorks" },
  { href: "#security", key: "landing.nav.security" },
  { href: "#use-cases", key: "landing.nav.useCases" },
  { href: "#faq", key: "landing.nav.faq" },
];

/**
 * SiteHeader: sticky, single line at desktop. Below 1024px the nav links and
 * Sign in collapse into a Radix Dialog hamburger menu; the locale switcher and
 * the single CTA stay in the bar at every size.
 */
export function SiteHeader() {
  const { t, locale, setLocale } = useI18n();
  const [menuOpen, setMenuOpen] = useState(false);
  return (
    <header
      style={{
        position: "sticky",
        top: 0,
        zIndex: 40,
        background: "color-mix(in srgb, var(--bg) 82%, transparent)",
        backdropFilter: "blur(14px)",
        WebkitBackdropFilter: "blur(14px)",
        borderBottom: "1px solid var(--line)",
      }}
    >
      <div
        className="container"
        style={{
          height: 68,
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: 16,
        }}
      >
        <a
          href="/landing"
          style={{ textDecoration: "none", display: "inline-flex", flexShrink: 0 }}
          aria-label="Skawld home"
        >
          <Logo />
        </a>
        <nav aria-label="Primary" className="nav-desktop">
          {NAV_LINKS.map((link) => (
            <a key={link.href} href={link.href}>
              {t(link.key)}
            </a>
          ))}
        </nav>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 10,
            marginLeft: "auto",
          }}
        >
          <select
            aria-label={t("lang.label")}
            value={locale}
            onChange={(event) => setLocale(event.target.value as Locale)}
            style={{
              appearance: "none",
              background: "transparent",
              border: "1px solid var(--line-soft)",
              borderRadius: 6,
              color: "var(--ink-soft)",
              fontSize: 13,
              fontWeight: 500,
              padding: "7px 10px",
              cursor: "pointer",
              flexShrink: 0,
            }}
          >
            <option value="en">EN</option>
            <option value="vi">VI</option>
          </select>
        </div>
        <div className="header-actions-desktop">
          <a
            href="/auth/login"
            className="btn btn-ghost"
            style={{ padding: "9px 16px", fontSize: 13 }}
          >
            {t("landing.signIn")}
          </a>
          <Button href="/auth/login">{t("landing.bookPilot")}</Button>
        </div>
        <div className="header-actions-mobile">
          <Button href="/auth/login" className="btn-compact">
            {t("landing.bookPilot")}
          </Button>
          <Dialog.Root open={menuOpen} onOpenChange={setMenuOpen}>
            <Dialog.Trigger asChild>
              <button
                type="button"
                aria-label={t("landing.nav.menu")}
                style={{
                  appearance: "none",
                  width: 40,
                  height: 40,
                  display: "grid",
                  placeItems: "center",
                  border: "1px solid var(--line-soft)",
                  borderRadius: 10,
                  background: "transparent",
                  color: "var(--ink)",
                  cursor: "pointer",
                  flexShrink: 0,
                }}
              >
                <List size={20} weight="bold" aria-hidden="true" />
              </button>
            </Dialog.Trigger>
            <Dialog.Portal>
              <Dialog.Content
                style={{
                  position: "fixed",
                  top: 68,
                  left: 0,
                  right: 0,
                  zIndex: 50,
                  background: "var(--surface)",
                  borderBottom: "1px solid var(--line)",
                  boxShadow: "0 24px 48px -24px rgba(0,0,0,0.5)",
                }}
              >
                <Dialog.Title className="visually-hidden">
                  {t("landing.nav.menu")}
                </Dialog.Title>
                <div className="container" style={{ display: "grid", gap: 4, paddingBlock: 12 }}>
                  {NAV_LINKS.map((link) => (
                    <a
                      key={link.href}
                      href={link.href}
                      onClick={() => setMenuOpen(false)}
                      style={{
                        color: "var(--ink-soft)",
                        fontSize: 15,
                        fontWeight: 500,
                        textDecoration: "none",
                        padding: "12px 4px",
                      }}
                    >
                      {t(link.key)}
                    </a>
                  ))}
                  <a
                    href="/auth/login"
                    onClick={() => setMenuOpen(false)}
                    style={{
                      color: "var(--ink-soft)",
                      fontSize: 15,
                      fontWeight: 500,
                      textDecoration: "none",
                      padding: "12px 4px",
                    }}
                  >
                    {t("landing.signIn")}
                  </a>
                </div>
              </Dialog.Content>
            </Dialog.Portal>
          </Dialog.Root>
        </div>
      </div>
    </header>
  );
}
```

Note: `Button` merges `className` (see `shared/Button.tsx:18`), so `className="btn-compact"` lands on the rendered `<a>` and `.btn-compact` overrides the pill padding because it appears later in `marketing.css`.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd web && npx vitest run src/marketing/Landing.test.tsx`
Expected: PASS — all 5 original tests (language switcher still resolves one select) + the 4 added across Tasks 2-4.

- [ ] **Step 7: Type-check and commit**

Run: `cd web && npm run lint`
Expected: no errors.

```bash
git add web/src/marketing/components/SiteHeader.tsx web/src/marketing/marketing.css web/src/test/setup.ts web/vite.config.ts web/src/marketing/Landing.test.tsx
git commit -m "fix(web): responsive header with hamburger menu for mobile"
```

---

### Task 5: Normalize the type hierarchy (bento + workflow rail)

**Files:**
- Modify: `web/src/marketing/components/FeaturesBento.tsx:59-69` (card title/body)
- Modify: `web/src/marketing/components/WorkflowRail.tsx:148` (step title)
- Test: `web/src/marketing/Landing.test.tsx` (extend)

**Interfaces:**
- Produces: bento card h3 at `fontSize: 18` with body at `fontSize: 14`; workflow step h3 at `fontSize: 20`. The e2e visual check (Task 7) relies on these exact values.

- [ ] **Step 1: Write the failing test**

Append to `web/src/marketing/Landing.test.tsx`:

```ts
  it("keeps card titles visibly above body text across sections", () => {
    renderLanding();
    const featuresTitles = Array.from(document.querySelectorAll("#features h3"));
    expect(featuresTitles.length).toBe(5);
    for (const h3 of featuresTitles) {
      expect((h3 as HTMLElement).style.fontSize).toBe("18px");
    }
    const workflowTitle = document.querySelector("#workflow h3") as HTMLElement;
    expect(workflowTitle.style.fontSize).toBe("20px");
  });
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/marketing/Landing.test.tsx`
Expected: FAIL — bento h3 is currently `16.5px`, workflow h3 `19px`.

- [ ] **Step 3: Fix the bento card scale**

In `web/src/marketing/components/FeaturesBento.tsx`, replace:

```tsx
      <h3 style={{ fontSize: 16.5, fontWeight: 650, color: "var(--ink)", margin: 0 }}>
        {title}
      </h3>
      <p
        style={{
          margin: 0,
          fontSize: 13.5,
          lineHeight: 1.6,
          color: "var(--ink-soft)",
        }}
      >
        {body}
      </p>
```

with:

```tsx
      <h3 style={{ fontSize: 18, fontWeight: 700, lineHeight: 1.3, color: "var(--ink)", margin: 0 }}>
        {title}
      </h3>
      <p
        style={{
          margin: 0,
          fontSize: 14,
          lineHeight: 1.6,
          color: "var(--ink-soft)",
        }}
      >
        {body}
      </p>
```

and change the cell gap from `gap: 12` to `gap: 14` in the cell container style (same `FeatureCell` component).

- [ ] **Step 4: Fix the workflow step title**

In `web/src/marketing/components/WorkflowRail.tsx`, replace:

```tsx
                <h3 style={{ fontSize: 19, fontWeight: 650, color: "var(--ink)", margin: 0 }}>
```

with:

```tsx
                <h3 style={{ fontSize: 20, fontWeight: 650, color: "var(--ink)", margin: 0 }}>
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd web && npx vitest run src/marketing/Landing.test.tsx`
Expected: PASS.

- [ ] **Step 6: Type-check and commit**

Run: `cd web && npm run lint`
Expected: no errors.

```bash
git add web/src/marketing/components/FeaturesBento.tsx web/src/marketing/components/WorkflowRail.tsx web/src/marketing/Landing.test.tsx
git commit -m "fix(web): normalize marketing type hierarchy across sections"
```

---

### Task 6: PilotCta grid backdrop, favicon, and openapi.yaml asset

**Files:**
- Modify: `web/src/marketing/components/PilotCta.tsx`
- Create: `web/public/favicon.svg`
- Modify: `web/index.html`
- Create: `web/public/openapi.yaml` (copy of `api/openapi.yaml`)
- Modify: `Makefile:39-40` (`web-check` refresh step)
- Test: `web/src/marketing/Landing.test.tsx` (extend)

**Interfaces:**
- Produces: `/favicon.svg` and `/openapi.yaml` served from `web/public` (vite serves public/ at root). Task 7's e2e asserts both return 200.

- [ ] **Step 1: Write the failing test**

Append to `web/src/marketing/Landing.test.tsx`:

```ts
  it("links the API spec to /openapi.yaml", () => {
    renderLanding();
    const specLink = screen.getByRole("link", { name: /read the api spec/i });
    expect(specLink.getAttribute("href")).toBe("/openapi.yaml");
  });
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npx vitest run src/marketing/Landing.test.tsx`
Expected: FAIL — no link matches `/read the api spec/i` (this also documents the existing copy; if the label changes, update the regex).

- [ ] **Step 3: Add the grid backdrop to PilotCta**

In `web/src/marketing/components/PilotCta.tsx`, change the `<section>` to `position: "relative"` and insert the backdrop as its first child:

```tsx
      <div
        aria-hidden="true"
        style={{
          position: "absolute",
          inset: 0,
          pointerEvents: "none",
          backgroundImage:
            "linear-gradient(var(--grid-line) 1px, transparent 1px), linear-gradient(90deg, var(--grid-line) 1px, transparent 1px)",
          backgroundSize: "56px 56px",
          maskImage:
            "radial-gradient(ellipse 70% 60% at 50% 100%, black 25%, transparent 75%)",
          WebkitMaskImage:
            "radial-gradient(ellipse 70% 60% at 50% 100%, black 25%, transparent 75%)",
        }}
      />
```

so the section opening becomes:

```tsx
    <section
      style={{
        position: "relative",
        paddingBlock: "clamp(72px, 9vw, 120px)",
        borderTop: "1px solid var(--line)",
        background:
          "linear-gradient(180deg, var(--bg) 0%, color-mix(in srgb, var(--brand) 6%, var(--bg)) 100%)",
      }}
    >
```

- [ ] **Step 4: Create the favicon**

Create `web/public/favicon.svg`:

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">
  <rect x="1" y="1" width="30" height="30" rx="8" fill="#101816" stroke="#2f6b4f" stroke-width="1.5"/>
  <text x="16" y="21.5" font-family="ui-monospace, monospace" font-size="15" font-weight="800" fill="#68d391" text-anchor="middle">S</text>
</svg>
```

In `web/index.html`, inside `<head>` after the `theme-color` line, add:

```html
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
```

- [ ] **Step 5: Serve openapi.yaml from the web app**

Run: `cp api/openapi.yaml web/public/openapi.yaml`

In `Makefile`, change the `web-check` target:

```make
web-check:
	cp api/openapi.yaml web/public/openapi.yaml
	cd web && npm ci && npm run lint && npm test && npm run build
```

(Exact tabs in the Makefile must be preserved; edit only the two lines shown.)

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd web && npx vitest run src/marketing/Landing.test.tsx`
Expected: PASS.

- [ ] **Step 7: Type-check and commit**

Run: `cd web && npm run lint`
Expected: no errors.

```bash
git add web/src/marketing/components/PilotCta.tsx web/public/favicon.svg web/index.html web/public/openapi.yaml Makefile web/src/marketing/Landing.test.tsx
git commit -m "fix(web): resolve landing 404s with favicon and served OpenAPI spec"
```

---

### Task 7: Playwright e2e — mobile overflow and asset regressions

**Files:**
- Modify: `web/package.json` (devDependency + script)
- Create: `web/playwright.config.ts`
- Create: `web/e2e/mobile-overflow.spec.ts`
- Create: `web/e2e/assets.spec.ts`

**Interfaces:**
- Consumes: `Menu`-labeled hamburger and `Features` nav link (Task 4), `/favicon.svg` + `/openapi.yaml` (Task 6), compact CTA `Book a pilot` (Task 4).
- Produces: `npm run test:e2e` script; the definitive regression gate for defects 1, 4, and 6.

- [ ] **Step 1: Install Playwright and add the script**

Run: `cd web && npm i -D @playwright/test`
Run: `cd web && npx playwright install chromium`
(If the browser install needs system libraries, run `npx playwright install --with-deps chromium`; this may require sudo on Linux.)

In `web/package.json` `scripts`, add:

```json
    "test:e2e": "playwright test"
```

- [ ] **Step 2: Create the Playwright config**

Create `web/playwright.config.ts`:

```ts
import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  use: {
    baseURL: "http://localhost:5173",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: {
    command: "npm run dev",
    url: "http://localhost:5173",
    reuseExistingServer: !process.env.CI,
  },
});
```

- [ ] **Step 3: Write the mobile overflow spec**

Create `web/e2e/mobile-overflow.spec.ts`:

```ts
import { test, expect } from "@playwright/test";

const horizontalOverflow = () =>
  `document.documentElement.scrollWidth - document.documentElement.clientWidth`;

test("landing page has no horizontal overflow at 390px, menu included", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/landing");
  await page.waitForLoadState("networkidle");
  expect(await page.evaluate(horizontalOverflow)).toBeLessThanOrEqual(0);

  const menu = page.getByRole("button", { name: "Menu" });
  await expect(menu).toBeVisible();
  await menu.click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  await expect(page.getByRole("link", { name: "Features" })).toBeVisible();
  expect(await page.evaluate(horizontalOverflow)).toBeLessThanOrEqual(0);
});

test("header CTA stays visible and tappable at 390px", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/landing");
  await expect(page.getByRole("link", { name: "Book a pilot" }).first()).toBeVisible();
});
```

- [ ] **Step 4: Write the asset and console-error spec**

Create `web/e2e/assets.spec.ts`:

```ts
import { test, expect } from "@playwright/test";

test("favicon and OpenAPI spec resolve without 404", async ({ request }) => {
  const favicon = await request.get("/favicon.svg");
  expect(favicon.status()).toBe(200);
  const openapi = await request.get("/openapi.yaml");
  expect(openapi.status()).toBe(200);
});

test("landing page loads without console errors at 390px", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") errors.push(msg.text());
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/landing");
  await page.waitForLoadState("networkidle");
  expect(errors).toEqual([]);
});
```

- [ ] **Step 5: Run the e2e suite**

Run: `cd web && npm run test:e2e`
Expected: PASS (4 tests). If any overflow or 404 fails, it is a regression from Tasks 2-6 — fix before committing.

- [ ] **Step 6: Run the full verification gate**

Run: `cd web && npm run lint && npm test`
Expected: lint clean, all vitest suites green (i18n, landing, console, presentation).

- [ ] **Step 7: Commit**

```bash
git add web/package.json web/package-lock.json web/playwright.config.ts web/e2e
git commit -m "test(web): add Playwright e2e for mobile overflow and asset 404s"
```

---

## Final acceptance checklist (run before declaring done)

- [ ] `cd web && npm run lint` — clean
- [ ] `cd web && npm test` — all suites pass
- [ ] `cd web && npm run test:e2e` — all 4 e2e tests pass
- [ ] `cd web && npm run build` — production build succeeds
- [ ] Manual dual-mode check (dark/light via `prefers-color-scheme`) and EN/VI toggle both render correctly
- [ ] No "P-302 · P-302" anywhere in the landing DOM
- [ ] Bento h3 = 18px, workflow h3 = 20px
- [ ] Header has exactly one locale select; hamburger menu opens with keyboard (Tab to trigger, Enter, ESC to close)

## Execution deviations (2026-08-06, recorded during Task 4/7)

1. **Header overflow root cause**: the e2e caught a residual 64px overflow at
   390px. The logo anchor was 205px wide (wordmark + "maintenance
   intelligence" tagline) and could not fit beside locale + CTA + hamburger.
   Fix: added `className` passthrough to `Logo.tsx`, and below 1024px
   `.logo-link .logo-wordmark { display: none }` collapses the lockup to the
   S tile (anchor aria-label keeps it accessible).
2. **vitest picked up `e2e/*.spec.ts`**: scoped vitest to
   `include: ["src/**/*.{test,spec}.{ts,tsx}"]` in `vite.config.ts` so the
   Playwright specs are only run by `npm run test:e2e`.

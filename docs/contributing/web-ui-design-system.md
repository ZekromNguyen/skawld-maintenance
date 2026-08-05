# Skawld Web UI Design System

Source of truth for building or changing **any UI in the skawld web console and
the public marketing page**. Read this before creating new UX/UI. It encodes
the taste-skill protocol applied to Skawld's brand, so agents can ship
consistent, premium, accessible, on-brand interfaces without re-deriving the
design.

---

## 0. Design Read

> Skawld is an **industrial maintenance intelligence layer**. The UI must read
> like a **calibrated instrument**: precise, trustworthy, dense where it needs
> to be, and premium. It is **not** a startup hype site, and it is **not** an
> artsy agency portfolio.

**Audiences:**

| Surface | Audience | Priority |
|---|---|---|
| Public marketing page (`/landing`) | Plant/reliability directors, ops leaders, CTOs, procurement | Trust-first premium |
| Operator console (root `/`) | Supervisors, senior techs, technicians, managers, admins | Speed, density, role-aware |

**One-line design read:**
*"B2B enterprise industrial SaaS, dark-tech precision language, Skawld's
instrument-emerald identity, Tailwind v4 + Geist/Geist Mono + restrained
motion, premium comparable to Linear."*

---

## 1. Taste Dials

Global dials. Every new page/component should be evaluated against these.

| Dial | Value | Meaning |
|---|---|---|
| `DESIGN_VARIANCE` | **6** | Offset, asymmetric split, fractional grids, one or two editorial moves. Not artsy chaos: industrial safety reads trust-first. |
| `MOTION_INTENSITY` | **5** | Fluid CSS only. Entry reveals, scroll-reveal stagger, hover physics on CTAs. No GSAP hijack, no marquee, no parallax. Every animation needs a reason (hierarchy / state / feedback). Safety products don't bounce. |
| `VISUAL_DENSITY` | **3** (marketing) / **8** (console) | Marketing: airy, editorial whitespace. Console: cockpit density, mono numerals, 1px hairlines over cards. |

---

## 2. Brand Tokens

### 2.1 Color (single locked accent: emerald)

The accent is **locked across the whole product**. Do not introduce a second
accent (no teal-on-one-page, no blue badge elsewhere).

**Dark (default identity, `prefers-color-scheme: dark`):**

| Token | Value | Use |
|---|---|---|
| `--bg` | `#0c120f` | Page background |
| `--surface` | `#101816` | Cards, panels |
| `--raised` | `#15201b` | Hover, elevated, table headers |
| `--sunken` | `#0a0f0d` | Wells, safety boundary |
| `--line` | `#223129` | Hairlines |
| `--line-soft` | `#2d3f35` | Input borders |
| `--ink` | `#e8efe9` | Primary text |
| `--ink-soft` | `#9fb0a4` | Secondary text |
| `--ink-muted` | `#667a6e` | Muted, captions |
| `--brand` | `#68d391` | Accent (emerald). ~8.9:1 on `--bg`, AA body-safe |
| `--brand-dim` | `#2f6b4f` | Brand borders, subtle fills |
| `--amber` | `#f0b35a` | Warnings |
| `--red` | `#e0665a` | Critical / destructive |
| `--blue` | `#7fb3e0` | Informational (sparingly) |

**Light mode:** same semantic tokens, deepened emerald accent family for AA
(approx `#1a7f4d` for text-on-light, `#68d391` kept for fills), green-tinted
neutral greys (zinc-family tinted green), never warm beige.

**Rules:**
- No pure `#000000`, no pure `#ffffff` (off-black / off-white only).
- No AI purple / violet / lila / neon gradient glow. Emerald is the only accent.
- One palette per project. Do not fluctuate warm/cool greys.
- Max 1 accent color per page.

### 2.2 Typography

- **Display / Body:** **Geist** (self-hosted WOFF2, `font-display: swap`).
- **Numerals / data / step labels:** **Geist Mono**.
- **Banned as default:** Inter, Fraunces, Instrument Serif.
- **Serif:** only for genuine editorial/luxury moments, and then not the
  LLM-favorite serifs. Default is sans-serif display.
- **Default type scale (marketing):** display `text-4xl md:text-5xl lg:text-6xl
  tracking-tighter leading-none`; body `text-base text-gray-600 (light) /
  text-gray-300 (dark) leading-relaxed max-w-[65ch]`.
- **Console:** compact: base 12-13px, mono numbers, `letter-spacing: .02em`
  on headings.

### 2.3 Shape lock (one radius system, documented)

| Element | Radius |
|---|---|
| Buttons (interactive) | Pill (full) |
| Cards / panels | 16px |
| Inputs | 10px |

Round buttons on a square-card page, or vice versa, is broken design. Pick one
system per surface and follow it everywhere.

### 2.4 Shadows

- Tint shadows to the background hue. No pure-black drop shadows on light
  backgrounds.
- Cards only where elevation communicates real hierarchy; otherwise use
  `border-t`, `divide-y`, or negative space.

---

## 3. Page Theme Lock

- One theme per session: dark, light, or `prefers-color-scheme` (default).
- No section flips to inverted theme mid-page.
- Both modes designed from the start; WCAG AA (AAA for hero copy); hierarchy
  parity across modes; brand accent stays recognizable in both.
- Console is dark-first brand identity; marketing respects system preference.

---

## 4. Motion Protocol

- **Only animate `transform` and `opacity`.** Never `top`, `left`, `width`,
  `height`.
- **Never use `window.addEventListener("scroll", ...)`.** Use Motion's
  `useScroll`, `whileInView`, GSAP ScrollTrigger, IntersectionObserver, or CSS
  scroll-driven animations.
- **`MOTION_INTENSITY > 3` must honor `prefers-reduced-motion`.** Collapse to
  static under reduced motion.
- Isolate motion in leaf client components with `useEffect` cleanup; use
  Motion `useMotionValue`/`useTransform` for continuous values, never
  `useState`.
- Micro-interactions: `:active` → `-translate-y-[1px]` or `scale-[0.98]`.
- Marquee: max one per page; only when the content justifies it.
- Sticky-stack / horizontal-pan (GSAP): `start: "top top"`, `pin: true`,
  `scrub` — per the canonical skeletons.

---

## 5. Conversion & Marketing Rules (landing and any public page)

- **One CTA intent per page.** Pick one primary intent label ("Book a pilot")
  and use it in nav, hero, and final band. Secondary intent ("Read the spec")
  at most once. No duplicate labels ("Get in touch" + "Contact us" + "Let's
  talk" = fail).
- **Hero discipline:** H1 max 2 lines desktop; subtext max 20 words; CTA
  visible without scroll; hero top padding max `pt-24`; max 4 text elements
  (eyebrow OR brand strip, headline, subtext, CTAs). No tiny taglines, no
  trust micro-strips, no version labels inside the hero.
- **Logo wall** lives UNDER the hero, logos only (no industry labels), real
  SVG logos (Simple Icons) or generated monogram marks. No plain text
  wordmarks.
- **Eyebrow budget:** max 1 eyebrow per 3 sections (`ceil(sections/3)`).
- **No duplicate CTA, no section-numbering eyebrows** (`00 / INDEX`),
  no `Scroll ↓` cues, no locale/weather strips, no decorative status dots
  (only real semantic state), no `border-t`+`border-b` on every table row.
- **Section-layout-repetition ban:** a page with 8+ sections must use at
  least 4 different layout families (split hero, bento, workflow rail, tabs,
  stat cards, accordion, quote cards, conversion band, logo wall, footer).
- **Bento grids:** exactly N cells for N items; no empty cells; at least 2-3
  cells carry real visual variation (image, tinted bg, live data component).
- **Spec sheets:** no divider-under-every-row tables. Use 2-col stat cards,
  scroll-snap pills, grouped chunks, or featured-vs-rest.
- **Testimonials:** quote ≤ 3 lines, real attribution (name + role), typo-
  graphic quotes, no em-dash in the quote.
- **Fake data:** real repo metrics OK; invented specs must be marked
  `<!-- sample -->` / "example". Never fake engineering precision the product
  doesn't claim.
- **Copy self-audit:** zero em-dashes (`—` / `–`) anywhere visible; no filler
  verbs ("Elevate", "Seamless", "Unleash"); no generic names (Jane Doe); no
  "Quietly trusted by"; one copy register per page.
- **Icons:** Phosphor / HugeIcons / Radix / Tabler only, one family per
  project, never hand-rolled SVG paths. Emoji discouraged.

---

## 6. Accessibility (non-negotiable)

- Skip-to-content link; landmarks (`header`, `nav`, `main`, `footer`).
- One `h1` per page; ordered heading hierarchy.
- Real `<button>` / `<a>` elements; visible `:focus-visible` rings.
- WCAG AA contrast on all text, form inputs, placeholders, focus rings, helper
  text, error text (4.5:1 body, 3:1 large).
- **Button contrast:** text readable against bg (no white-on-white, no
  transparent ghost over photo without scrim).
- **CTA wrap ban:** no CTA label wraps to 2+ lines at desktop. Shorten label
  or widen button.
- Labels ABOVE inputs, error text BELOW, no placeholder-as-label.
- Status conveyed with text, not color alone (severity badges carry a label).
- Keyboard-nav for tabs, accordion, dialogs; `aria-*` on icon-only controls.

---

## 7. Performance (Core Web Vitals)

- LCP < 2.5s: hero image `preload`/priority or lightweight DOM/CSS visual;
  self-hosted fonts with `font-display: swap` + display-face preload.
- INP < 200ms: heavy work off main thread; Motion values, not React state.
- CLS < 0.1: reserve space for images, fonts, accordions.
- Grain/noise overlays only on fixed `pointer-events-none` pseudo-elements,
  never on scrolling containers.
- `min-h-[100dvh]`, never `h-screen`.
- `max-w-[1400px] mx-auto` or `max-w-7xl` containment.
- Grid over flex-math: `grid grid-cols-1 md:grid-cols-3 gap-6`, no
  `w-[calc(33%-1rem)]`.
- Mobile collapse explicit per section (`w-full px-4` below `md`).

---

## 8. Console (Operator UI) Rules

The console is a **role-aware, data-dense product UI**. Marketing aesthetics
do not apply to it.

- **Stack:** shadcn/ui (or Radix Themes) primitives + Tailwind v4 +
  Phosphor/Tabler icons. Owned, accessible components; never default state
  (customize radii/colors to brand tokens above).
- **Density:** cockpit. 1px hairlines over cards; `font-mono` for numbers;
  compact padding (12-13px text).
- **Role-aware:** gate nav items and action buttons by
  `principal.permissions`. A Technician never sees "Publish workflow"; a
  Supervisor never sees "Create organization". One shell, branching by role.
- **Per-role landing:** Supervisor → open-incident queue; Technician →
  assigned executions; Manager → team handover; Administrator → org health.
- **State coverage:** skeleton loaders (not generic spinners), composed empty
  states with CTAs, contextual error toasts, optimistic updates on safe
  actions.
- **Keep the instrument theme:** existing console CSS (`styles.css`) is the
  brand. Port to CSS-variable tokens, do not throw it out.
- **Not for the console:** hero sections, marquees, landing-page motion,
  marketing eyebrows, centered editorial layouts.

---

## 9. Structure & Conventions

- **Marketing page:** `web/src/marketing/` (`Landing.tsx`, `marketing.css`,
  `data.ts`, `components/`). Route: `/landing` (pathname gate in
  `main.tsx`; console stays at root `/`).
- **Console:** `web/src/App.tsx`, `web/src/styles.css`, `web/src/i18n/`.
- En/vi i18n for the console; public marketing page is English-first.
- New pages must render in both themes and pass the pre-flight checklist
  before being called done.

---

## 10. Pre-Flight Checklist (run before declaring any UI done)

- [ ] Design read declared (Section 0) and dials explicit (Section 1)
- [ ] ZERO em-dashes (`—` / `–`) anywhere visible
- [ ] One accent color, locked across the whole page
- [ ] One radius system, applied consistently
- [ ] Theme lock: one theme, no mid-page inversion
- [ ] Button + form contrast AA; no CTA wrap at desktop
- [ ] Hero fits viewport (≤2-line H1, ≤20-word subtext, CTA visible)
- [ ] Eyebrow count ≤ `ceil(sections/3)`
- [ ] No duplicate CTA intent on the page
- [ ] No section-numbering eyebrows, no scroll cues, no locale strips
- [ ] Logo wall under hero, logos only
- [ ] Bento: N items → N cells, 2-3 visually varied cells
- [ ] Section-layout families ≥ 4 for 8+ section pages
- [ ] Long lists use the right UI component (not bare `divide-y` rows)
- [ ] Real images or honest placeholders; no div-fake-screenshots
- [ ] Copy self-audit: no filler verbs, no fake precision, one register
- [ ] Motion motivated, reduced-motion honored, transform/opacity only
- [ ] No `window.addEventListener("scroll")`
- [ ] Mobile collapse explicit; `min-h-[100dvh]`; grid not flex-math
- [ ] Empty / loading / error states present
- [ ] Icons from allowed library, one family
- [ ] LCP < 2.5s, CLS < 0.1 plausibly met
- [ ] Rendered and verified in both light and dark mode

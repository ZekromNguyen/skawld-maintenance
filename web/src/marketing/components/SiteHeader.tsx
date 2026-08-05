import { Logo } from "./shared/Logo";
import { Button } from "./shared/Button";

const NAV_LINKS = [
  { href: "#features", label: "Features" },
  { href: "#workflow", label: "How it works" },
  { href: "#security", label: "Security" },
  { href: "#use-cases", label: "Use cases" },
  { href: "#faq", label: "FAQ" },
];

/**
 * SiteHeader: sticky, single line at desktop, max 72px.
 * Primary CTA intent: "Book a pilot" (one label across the whole page).
 */
export function SiteHeader() {
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
          gap: 24,
        }}
      >
        <a
          href="/landing"
          style={{ textDecoration: "none", display: "inline-flex" }}
          aria-label="Skawld home"
        >
          <Logo />
        </a>
        <nav aria-label="Primary" className="nav-desktop">
          {NAV_LINKS.map((link) => (
            <a key={link.href} href={link.href}>
              {link.label}
            </a>
          ))}
        </nav>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <a
            href="/auth/login"
            className="btn btn-ghost"
            style={{ padding: "9px 16px", fontSize: 13 }}
          >
            Sign in
          </a>
          <Button href="/auth/login">Book a pilot</Button>
        </div>
      </div>
      <div className="container nav-mobile" aria-label="Secondary">
        {NAV_LINKS.map((link) => (
          <a key={link.href} href={link.href}>
            {link.label}
          </a>
        ))}
      </div>
    </header>
  );
}

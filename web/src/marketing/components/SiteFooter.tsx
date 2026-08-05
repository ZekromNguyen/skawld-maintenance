import { Logo } from "./shared/Logo";

/**
 * SiteFooter: docs, GitHub, contact, legal. Functional links, no decoration.
 */
const COLUMNS = [
  {
    heading: "Product",
    links: [
      { label: "Features", href: "#features" },
      { label: "How it works", href: "#workflow" },
      { label: "Security", href: "#security" },
      { label: "Use cases", href: "#use-cases" },
      { label: "FAQ", href: "#faq" },
    ],
  },
  {
    heading: "Documentation",
    links: [
      { label: "Specification", href: "/spec" },
      { label: "OpenAPI contract", href: "/openapi.yaml" },
      { label: "Runbooks", href: "/docs" },
      { label: "SDK", href: "https://github.com/ZekromNguyen/skawld-sdk-go" },
    ],
  },
  {
    heading: "Contact",
    links: [
      { label: "GitHub", href: "https://github.com/ZekromNguyen/skawld-maintenance" },
      { label: "Contact us", href: "mailto:hello@skawld.dev" },
      { label: "Status", href: "/health/live" },
    ],
  },
];

export function SiteFooter() {
  return (
    <footer
      style={{
        borderTop: "1px solid var(--line)",
        background: "var(--sunken)",
        paddingBlock: "56px 40px",
      }}
    >
      <div className="container">
        <div className="footer-grid">
          <div>
            <Logo />
            <p
              style={{
                marginTop: 16,
                color: "var(--ink-muted)",
                fontSize: 13.5,
                lineHeight: 1.6,
                maxWidth: "36ch",
              }}
            >
              The maintenance intelligence layer that captures expertise,
              preserves safety, and keeps human authority in control.
            </p>
          </div>
          <div className="footer-links">
            {COLUMNS.map((column) => (
              <nav key={column.heading} aria-label={column.heading}>
                <h3
                  style={{
                    margin: "0 0 14px",
                    fontSize: 11,
                    fontFamily: '"Geist Mono Variable", monospace',
                    letterSpacing: "0.14em",
                    textTransform: "uppercase",
                    color: "var(--ink-muted)",
                  }}
                >
                  {column.heading}
                </h3>
                <ul style={{ listStyle: "none", margin: 0, padding: 0, display: "grid", gap: 10 }}>
                  {column.links.map((link) => (
                    <li key={link.label}>
                      <a
                        href={link.href}
                        style={{
                          color: "var(--ink-soft)",
                          fontSize: 14,
                          textDecoration: "none",
                        }}
                      >
                        {link.label}
                      </a>
                    </li>
                  ))}
                </ul>
              </nav>
            ))}
          </div>
        </div>
        <div
          style={{
            marginTop: 44,
            paddingTop: 22,
            borderTop: "1px solid var(--line)",
            display: "flex",
            flexWrap: "wrap",
            gap: 16,
            justifyContent: "space-between",
            color: "var(--ink-muted)",
            fontSize: 12.5,
          }}
        >
          <span>© {new Date().getFullYear()} Skawld. All rights reserved.</span>
          <div style={{ display: "flex", gap: 18 }}>
            <a href="#" style={{ color: "inherit", textDecoration: "none" }}>
              Privacy
            </a>
            <a href="#" style={{ color: "inherit", textDecoration: "none" }}>
              Terms
            </a>
            <a href="#" style={{ color: "inherit", textDecoration: "none" }}>
              Security
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}

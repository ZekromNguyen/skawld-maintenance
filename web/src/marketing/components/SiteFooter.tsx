import { Logo } from "./shared/Logo";
import { useI18n } from "../../i18n/I18nProvider";
import type { MessageKey } from "../../i18n/messages";

/**
 * SiteFooter: docs, GitHub, contact, legal. Functional links, no decoration.
 */
const COLUMNS: Array<{
  headingKey: MessageKey;
  links: Array<{ labelKey: MessageKey; href: string }>;
}> = [
  {
    headingKey: "landing.footer.product",
    links: [
      { labelKey: "landing.nav.features", href: "#features" },
      { labelKey: "landing.nav.howItWorks", href: "#workflow" },
      { labelKey: "landing.nav.security", href: "#security" },
      { labelKey: "landing.nav.useCases", href: "#use-cases" },
      { labelKey: "landing.nav.faq", href: "#faq" },
    ],
  },
  {
    headingKey: "landing.footer.documentation",
    links: [
      { labelKey: "landing.footer.specification", href: "/spec" },
      { labelKey: "landing.footer.openapi", href: "/openapi.yaml" },
      { labelKey: "landing.footer.runbooks", href: "/docs" },
      { labelKey: "landing.footer.sdk", href: "https://github.com/ZekromNguyen/skawld-sdk-go" },
    ],
  },
  {
    headingKey: "landing.footer.contact",
    links: [
      { labelKey: "landing.footer.github", href: "https://github.com/ZekromNguyen/skawld-maintenance" },
      { labelKey: "landing.footer.contactUs", href: "mailto:hello@skawld.dev" },
      { labelKey: "landing.footer.status", href: "/health/live" },
    ],
  },
];

export function SiteFooter() {
  const { t } = useI18n();
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
              {t("landing.footer.description")}
            </p>
          </div>
          <div className="footer-links">
            {COLUMNS.map((column) => (
              <nav key={column.headingKey} aria-label={t(column.headingKey)}>
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
                  {t(column.headingKey)}
                </h3>
                <ul style={{ listStyle: "none", margin: 0, padding: 0, display: "grid", gap: 10 }}>
                  {column.links.map((link) => (
                    <li key={link.href}>
                      <a
                        href={link.href}
                        style={{
                          color: "var(--ink-soft)",
                          fontSize: 14,
                          textDecoration: "none",
                        }}
                      >
                        {t(link.labelKey)}
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
          <span>{t("landing.footer.rights", { year: new Date().getFullYear() })}</span>
          <div style={{ display: "flex", gap: 18 }}>
            <a href="#" style={{ color: "inherit", textDecoration: "none" }}>
              {t("landing.footer.privacy")}
            </a>
            <a href="#" style={{ color: "inherit", textDecoration: "none" }}>
              {t("landing.footer.terms")}
            </a>
            <a href="#" style={{ color: "inherit", textDecoration: "none" }}>
              {t("landing.footer.security")}
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}

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

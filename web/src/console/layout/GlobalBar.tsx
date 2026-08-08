import { useState } from "react";
import { MagnifyingGlass, Moon, Sun } from "@phosphor-icons/react";
import { useI18n } from "../../i18n/I18nProvider";
import { useTheme } from "../../theme/ThemeProvider";
import { useSite } from "../state/SiteContext";
import { SiteSwitcher } from "./SiteSwitcher";
import { CommandPalette } from "../components/CommandPalette";
import type { Principal } from "../../types";
import type { Locale } from "../../i18n/messages";

/**
 * GlobalBar: persistent console chrome above the page content. Holds the
 * command-palette trigger, global search affordance, site switcher, theme
 * and language switches, and operator presence. Rendered once by
 * ConsoleLayout so pages no longer duplicate chrome per route.
 */
export function GlobalBar({ principal }: { principal?: Principal }) {
  const { t, locale, setLocale } = useI18n();
  const { theme, toggleTheme } = useTheme();
  const { siteId, setSiteId } = useSite();
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
          <span>{t("cmd.placeholder")}</span>
          <kbd>/</kbd>
        </button>
        <div className="global-bar-right">
          <SiteSwitcher siteIds={principal?.site_ids ?? []} value={siteId} onChange={setSiteId} />
          <button
            type="button"
            className="global-icon-button"
            onClick={toggleTheme}
            aria-label={theme === "light" ? t("theme.switchToDark") : t("theme.switchToLight")}
          >
            {theme === "light" ? <Moon size={14} aria-hidden="true" /> : <Sun size={14} aria-hidden="true" />}
          </button>
          <label className="global-lang">
            <span className="visually-hidden">{t("lang.label")}</span>
            <select value={locale} onChange={(event) => setLocale(event.target.value as Locale)}>
              <option value="en">EN</option>
              <option value="vi">VI</option>
            </select>
          </label>
          <div className="operator">
            <span className="presence" />
            <span>
              <strong>{principal?.display_name ?? t("topbar.connecting")}</strong>
              <small>
                {principal?.site_ids.length
                  ? t("topbar.siteScope", { count: principal.site_ids.length })
                  : t("topbar.allSiteScope")}
              </small>
            </span>
          </div>
        </div>
      </header>
      <CommandPalette principal={principal} open={paletteOpen} onOpenChange={setPaletteOpen} />
    </>
  );
}

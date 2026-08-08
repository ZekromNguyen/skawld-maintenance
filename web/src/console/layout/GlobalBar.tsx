import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import {
  CaretDown,
  Cube,
  GearSix,
  Moon,
  SignOut,
  Sun,
  User,
  WarningCircle,
} from "@phosphor-icons/react";
import { useI18n } from "../../i18n/I18nProvider";
import { useTheme } from "../../theme/ThemeProvider";
import { useSite } from "../state/SiteContext";
import { SiteSwitcher } from "./SiteSwitcher";
import { CommandPalette } from "../components/CommandPalette";
import type { Principal } from "../../types";
import type { Locale } from "../../i18n/messages";
import { can } from "../permissions";

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  const first = parts[0][0] ?? "?";
  const last = parts.length > 1 ? parts[parts.length - 1][0] ?? "" : "";
  return (first + last).toUpperCase();
}

function signOut() {
  const form = document.createElement("form");
  form.method = "POST";
  form.action = "/auth/logout";
  form.style.display = "none";
  document.body.appendChild(form);
  form.submit();
}

/**
 * GlobalBar: Jira-style top chrome. Brand at left, palette trigger centered,
 * and at right the site/theme/language controls, a permission-gated Create
 * menu, and the avatar account menu.
 */
export function GlobalBar({ principal }: { principal?: Principal }) {
  const { t, locale, setLocale } = useI18n();
  const { theme, toggleTheme } = useTheme();
  const { siteId, setSiteId } = useSite();
  const navigate = useNavigate();
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [accountOpen, setAccountOpen] = useState(false);
  const createRef = useRef<HTMLDivElement>(null);
  const accountRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function onPointerDown(event: MouseEvent) {
      const target = event.target as Node;
      if (createRef.current && !createRef.current.contains(target)) setCreateOpen(false);
      if (accountRef.current && !accountRef.current.contains(target)) setAccountOpen(false);
    }
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setCreateOpen(false);
        setAccountOpen(false);
      }
    }
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, []);

  const canIncident = can(principal, "incident:create");
  const canAsset = can(principal, "asset:create");

  return (
    <>
      <header className="global-bar">
        <div className="brand">
          <span className="brand-mark">S</span>
          <span>
            <strong>Skawld</strong>
            <small>{t("topbar.maintenanceOps")}</small>
          </span>
        </div>
        <button
          type="button"
          className="global-command"
          onClick={() => setPaletteOpen(true)}
        >
          <span>{t("cmd.placeholder")}</span>
          <kbd>/</kbd>
        </button>
        <div className="global-bar-right">
          <SiteSwitcher siteIds={principal?.site_ids ?? []} value={siteId} onChange={setSiteId} />
          <label className="global-lang">
            <span className="visually-hidden">{t("lang.label")}</span>
            <select value={locale} onChange={(event) => setLocale(event.target.value as Locale)}>
              <option value="en">EN</option>
              <option value="vi">VI</option>
            </select>
          </label>
          {canIncident || canAsset ? (
            <div className="menu-anchor" ref={createRef}>
              <button
                type="button"
                className="primary-button create-button"
                aria-haspopup="menu"
                aria-expanded={createOpen}
                onClick={() => setCreateOpen((open) => !open)}
              >
                {t("topbar.create")}
                <CaretDown size={12} aria-hidden="true" />
              </button>
              {createOpen ? (
                <div className="menu-pop" role="menu">
                  {canIncident ? (
                    <button
                      type="button"
                      role="menuitem"
                      className="menu-item"
                      onClick={() => {
                        setCreateOpen(false);
                        navigate("/incidents?create=1");
                      }}
                    >
                      <WarningCircle size={14} aria-hidden="true" />
                      {t("topbar.createNewIncident")}
                    </button>
                  ) : null}
                  {canAsset ? (
                    <button
                      type="button"
                      role="menuitem"
                      className="menu-item"
                      onClick={() => {
                        setCreateOpen(false);
                        navigate("/assets?create=1");
                      }}
                    >
                      <Cube size={14} aria-hidden="true" />
                      {t("topbar.createNewAsset")}
                    </button>
                  ) : null}
                </div>
              ) : null}
            </div>
          ) : null}
          <button
            type="button"
            className="global-icon-button"
            onClick={toggleTheme}
            aria-label={theme === "light" ? t("theme.switchToDark") : t("theme.switchToLight")}
          >
            {theme === "light" ? <Moon size={14} aria-hidden="true" /> : <Sun size={14} aria-hidden="true" />}
          </button>
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
          <div className="menu-anchor" ref={accountRef}>
            <button
              type="button"
              className="avatar-button"
              aria-haspopup="menu"
              aria-expanded={accountOpen}
              aria-label={t("menu.profile")}
              onClick={() => setAccountOpen((open) => !open)}
            >
              {initials(principal?.display_name ?? "")}
            </button>
            {accountOpen ? (
              <div className="menu-pop menu-pop-wide" role="menu">
                <div className="menu-head">
                  <span className="avatar-button avatar-button-large" aria-hidden="true">
                    {initials(principal?.display_name ?? "")}
                  </span>
                  <span>
                    <strong>{principal?.display_name ?? t("topbar.connecting")}</strong>
                    <small>
                      {principal?.roles?.length
                        ? principal.roles.join(", ")
                        : t("authz.noRoles")}
                    </small>
                  </span>
                </div>
                <button
                  type="button"
                  role="menuitem"
                  className="menu-item"
                  onClick={() => {
                    setAccountOpen(false);
                    navigate("/account");
                  }}
                >
                  <User size={14} aria-hidden="true" />
                  {t("menu.profile")}
                </button>
                <button
                  type="button"
                  role="menuitem"
                  className="menu-item"
                  onClick={() => {
                    setAccountOpen(false);
                    toggleTheme();
                  }}
                >
                  <GearSix size={14} aria-hidden="true" />
                  {t("menu.theme")}
                </button>
                <button type="button" role="menuitem" className="menu-item" onClick={signOut}>
                  <SignOut size={14} aria-hidden="true" />
                  {t("nav.signOut")}
                </button>
              </div>
            ) : null}
          </div>
        </div>
      </header>
      <CommandPalette principal={principal} open={paletteOpen} onOpenChange={setPaletteOpen} />
    </>
  );
}

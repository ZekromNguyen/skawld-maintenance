import { Link } from "react-router";
import { Keyboard, type Icon } from "@phosphor-icons/react";
import { useI18n } from "../../i18n/I18nProvider";

/**
 * ShortcutCard: a navigable card linking into a page from the dashboard, or
 * a keyboard-shortcut legend when rendered with variant="shortcuts".
 */
export function ShortcutCard({
  icon,
  label,
  detail,
  to,
  variant = "navigate",
}: {
  icon?: Icon;
  label: string;
  detail: string;
  to: string;
  variant?: "navigate" | "shortcuts";
}) {
  const { t } = useI18n();
  if (variant === "shortcuts") {
    return (
      <section className="shortcut-card shortcut-card--legend" aria-label={t("shortcuts.title")}>
        <span className="shortcut-icon">
          <Keyboard weight="duotone" size={20} />
        </span>
        <span>
          <strong>{t("shortcuts.title")}</strong>
          <ul className="shortcut-legend">
            <li>
              <span>{t("shortcuts.palette")}</span>
              <kbd>{t("shortcuts.paletteKeys")}</kbd>
            </li>
            <li>
              <span>{t("shortcuts.navRows")}</span>
              <kbd>{t("shortcuts.navRowsKeys")}</kbd>
            </li>
            <li>
              <span>{t("shortcuts.openRow")}</span>
              <kbd>{t("shortcuts.openRowKeys")}</kbd>
            </li>
          </ul>
        </span>
      </section>
    );
  }
  const Icon = icon as Icon;
  return (
    <Link to={to} className="shortcut-card">
      <span className="shortcut-icon">
        <Icon weight="duotone" size={20} />
      </span>
      <span>
        <strong>{label}</strong>
        <small>{detail}</small>
      </span>
    </Link>
  );
}

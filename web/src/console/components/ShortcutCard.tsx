import { Link } from "react-router";
import type { Icon } from "@phosphor-icons/react";

/**
 * ShortcutCard: a navigable card linking into a page from the dashboard.
 */
export function ShortcutCard({
  icon,
  label,
  detail,
  to
}: {
  icon: Icon;
  label: string;
  detail: string;
  to: string;
}) {
  const Icon = icon;
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

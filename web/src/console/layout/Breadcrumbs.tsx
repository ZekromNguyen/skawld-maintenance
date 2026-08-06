import { Link } from "react-router";

export type TrailItem = { label: string; to?: string };

/**
 * Breadcrumbs: trailing navigation trail (e.g. Incidents / IN-1042).
 */
export function Breadcrumbs({ trail }: { trail: TrailItem[] }) {
  return (
    <nav aria-label="Breadcrumb" className="breadcrumbs">
      {trail.map((item, index) => (
        <span
          key={index}
          style={{ display: "inline-flex", alignItems: "center", gap: 8 }}
        >
          {index > 0 && (
            <span className="sep" aria-hidden="true">
              /
            </span>
          )}
          {item.to ? <Link to={item.to}>{item.label}</Link> : <span>{item.label}</span>}
        </span>
      ))}
    </nav>
  );
}

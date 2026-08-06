import { Link } from "react-router-dom";
import type { Tone } from "./StatusBadge";

export function MetricCard({
  label,
  value,
  detail,
  tone = "info",
  to,
  onClick,
}: {
  label: string;
  value: string;
  detail?: string;
  tone?: Tone;
  to?: string;
  onClick?: () => void;
}) {
  const body = (
    <div className={`metric metric-${tone}`}>
      <span>{label}</span>
      <strong>{value}</strong>
      {detail ? <small>{detail}</small> : null}
    </div>
  );
  if (to) {
    return (
      <Link to={to} className="metric-link">
        {body}
      </Link>
    );
  }
  if (onClick) {
    return (
      <button type="button" className="metric-link metric-button" onClick={onClick}>
        {body}
      </button>
    );
  }
  return body;
}

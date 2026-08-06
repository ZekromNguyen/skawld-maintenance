import { Link } from "react-router-dom";
import type { Tone } from "./StatusBadge";

export function MetricCard({
  label,
  value,
  detail,
  tone = "info",
  to,
}: {
  label: string;
  value: string;
  detail?: string;
  tone?: Tone;
  to?: string;
}) {
  const body = (
    <div className={`metric metric-${tone}`}>
      <span>{label}</span>
      <strong>{value}</strong>
      {detail ? <small>{detail}</small> : null}
    </div>
  );
  return to ? (
    <Link to={to} className="metric-link">
      {body}
    </Link>
  ) : (
    body
  );
}

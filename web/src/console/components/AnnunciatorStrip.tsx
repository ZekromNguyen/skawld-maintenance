import { Link } from "react-router";
import type { Tone } from "../ui/StatusBadge";

export interface AnnunciatorCell {
  key: string;
  label: string;
  value: string;
  tone: Tone;
  to?: string;
  eyebrow?: string;
}

/**
 * AnnunciatorStrip: control-room style readout of live operational counts.
 * Each cell is severity-coded on its left edge and links through when a
 * target route is given. Numerals use tabular figures so counts never jitter.
 */
export function AnnunciatorStrip({ cells }: { cells: AnnunciatorCell[] }) {
  return (
    <section className="annunciator" role="list" aria-label="Operational status">
      {cells.map((cell) => {
        const body = (
          <div className={`annunciator-cell annunciator-${cell.tone}`}>
            <span className="annunciator-label">{cell.label}</span>
            <strong className="mono-num">{cell.value}</strong>
            {cell.eyebrow ? <small className="annunciator-eyebrow">{cell.eyebrow}</small> : null}
          </div>
        );
        return (
          <div key={cell.key} role="listitem">
            {cell.to ? (
              <Link to={cell.to} className="annunciator-link" aria-label={cell.label}>
                {body}
              </Link>
            ) : (
              body
            )}
          </div>
        );
      })}
    </section>
  );
}

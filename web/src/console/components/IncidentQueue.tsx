import { Link } from "react-router-dom";
import { useI18n } from "../../i18n/I18nProvider";
import { relativeTime, severityTone } from "../../presentation";
import type { Incident } from "../../types";

/**
 * IncidentQueue: prioritized incident list; each row links to its detail.
 */
export function IncidentQueue({
  incidents,
  onCreate
}: {
  incidents: Incident[];
  onCreate: () => void;
}) {
  const { t, locale } = useI18n();
  return (
    <section className="queue panel">
      <div className="panel-heading">
        <div><span className="eyebrow">{t("incidents.authorizedScope")}</span><h2>{t("incidents.queue")}</h2></div>
        <div className="heading-actions"><span className="count">{incidents.length}</span><button className="secondary-button" onClick={onCreate}>{t("incidents.new")}</button></div>
      </div>
      <div className="queue-list">
        {incidents.map((incident) => (
          <Link
            key={incident.id}
            to={`/incidents/${incident.id}`}
            className="queue-item"
            style={{ textDecoration: "none", display: "grid", gridTemplateColumns: "4px 1fr", gap: 11, width: "100%", border: 0, borderBottom: "1px solid var(--line)", background: "var(--surface)", padding: "13px 14px", textAlign: "left", color: "inherit" }}
          >
            <span className={`severity-bar ${severityTone(incident.severity)}`} />
            <span>
              <span className="queue-meta"><span className="mono">{incident.number}</span><span>{relativeTime(incident.detected_at, locale)}</span></span>
              <strong style={{ display: "block", fontSize: 12, lineHeight: 1.45, margin: "6px 0 4px" }}>{incident.asset_tag} · {incident.summary}</strong>
              <small>{incident.state.replace("_", " ")} · {incident.severity}</small>
            </span>
          </Link>
        ))}
        {incidents.length === 0 && <div className="empty">{t("incidents.none")}</div>}
      </div>
    </section>
  );
}

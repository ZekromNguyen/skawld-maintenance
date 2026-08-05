import { useI18n } from "../../i18n/I18nProvider";
import { relativeTime, severityTone } from "../../presentation";
import type { Incident } from "../../types";
import { EmptyRow } from "./EmptyRow";

export function IncidentTable({ incidents }: { incidents: Incident[] }) {
  const { t, locale } = useI18n();
  return (
    <div className="table-wrap">
      <table>
        <thead><tr><th>{t("incidents.incident")}</th><th>{t("assets.asset")}</th><th>{t("incidents.summary")}</th><th>{t("incidents.severity")}</th><th>{t("incidents.state")}</th><th>{t("incidents.detected")}</th></tr></thead>
        <tbody>
          {incidents.map((incident) => (
            <tr key={incident.id}>
              <td className="mono strong">{incident.number}</td>
              <td className="mono">{incident.asset_tag || "—"}</td>
              <td className="summary-cell">{incident.summary}</td>
              <td><span className={`severity ${severityTone(incident.severity)}`}>{incident.severity}</span></td>
              <td>{incident.state.replace("_", " ")}</td>
              <td>{relativeTime(incident.detected_at, locale)}</td>
            </tr>
          ))}
          {incidents.length === 0 && <EmptyRow columns={6} labelKey="incidents.empty" />}
        </tbody>
      </table>
    </div>
  );
}

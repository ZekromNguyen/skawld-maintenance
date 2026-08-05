import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useApi } from "../useApi";
import { usePrincipal } from "../usePrincipal";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import type { Step } from "../../types";
import { Topbar } from "../layout/Topbar";
import { Breadcrumbs } from "../layout/Breadcrumbs";
import { MeasurementForm } from "../components/MeasurementForm";
import { StepRail } from "../components/StepRail";

/**
 * ExecutionWorkbenchPage: the pilot anchor page. LOTO step rail, live
 * measurements, evidence, and report drafting with state transitions.
 */
export function ExecutionWorkbenchPage() {
  const { executionId } = useParams<{ executionId: string }>();
  const { t } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const execution = useApi(() => api.execution(executionId!));
  const [busy, setBusy] = useState(false);

  const canWrite = principal?.permissions.includes("execution:write") ?? false;

  if (execution.error) {
    return <div className="toast-error" role="alert">{execution.error}</div>;
  }
  if (execution.loading || !execution.data) {
    return <div className="skeleton" style={{ height: 300 }} />;
  }
  const value = execution.data;

  const mutate = async (action: () => Promise<unknown>) => {
    setBusy(true);
    try {
      await action();
      await execution.refetch();
    } finally {
      setBusy(false);
    }
  };

  const toggleStep = (step: Step) => {
    if (!canWrite || step.state === "BLOCKED") return;
    void mutate(() => api.completeStep(value, step));
  };

  const start = () => void mutate(() => api.startExecution(value));

  const draftReport = () =>
    void mutate(async () => {
      const report = await api.draftReport(value.id);
      navigate(`/reports/${report.id}`);
    });

  return (
    <section>
      <Breadcrumbs
        trail={[
          { label: t("nav.incidents"), to: "/incidents" },
          { label: value.asset_tag }
        ]}
      />
      <Topbar title={value.purpose} principal={principal} />
      <div style={{ display: "grid", gap: 16 }}>
        <div className="panel">
          <div className="panel-heading">
            <div>
              <span className="eyebrow">{t("execution.deterministicExecution")}</span>
              <h2>{value.purpose}</h2>
            </div>
            <span className="state-badge">{value.state}</span>
          </div>
          <div style={{ padding: 16, display: "flex", gap: 10, flexWrap: "wrap" }}>
            {value.state === "ASSIGNED" && canWrite && (
              <button className="primary-button" disabled={busy} onClick={start}>
                {t("execution.startInspection")}
              </button>
            )}
            <button className="secondary-button" disabled={busy} onClick={draftReport}>
              {t("execution.draftReport")}
            </button>
          </div>
        </div>
        <StepRail steps={value.steps} onToggle={toggleStep} />
        <div style={{ display: "grid", gap: 16, gridTemplateColumns: "minmax(0,1fr)" }}>
          <div className="panel">
            <div className="panel-heading"><h2>{t("execution.measurements")}</h2></div>
            <div style={{ padding: 16 }}>
              {canWrite && (
                <MeasurementForm
                  busy={busy}
                  onSubmit={(type, measure, unit) =>
                    mutate(() => api.recordMeasurement(value, type, measure, unit))
                  }
                />
              )}
              <div style={{ display: "grid", gap: 8, marginTop: 12 }}>
                {value.measurements.map((measurement) => (
                  <div key={measurement.id} style={{ display: "flex", gap: 12, fontSize: 12 }}>
                    <span className="mono strong">{measurement.value}{measurement.unit}</span>
                    <span>{measurement.measurement_type}</span>
                    <span style={{ color: "var(--ink-muted)" }}>{measurement.observed_at}</span>
                  </div>
                ))}
                {value.measurements.length === 0 && (
                  <div className="empty">{t("execution.noMeasurements")}</div>
                )}
              </div>
            </div>
          </div>
          <div className="panel">
            <div className="panel-heading"><h2>{t("execution.observations")}</h2></div>
            <div style={{ padding: 16, display: "grid", gap: 8 }}>
              {value.observations.length > 0 ? (
                value.observations.map((observation, index) => (
                  <div key={index} style={{ fontSize: 12, color: "var(--ink-soft)" }}>
                    {JSON.stringify(observation)}
                  </div>
                ))
              ) : (
                <div className="empty">{t("execution.noObservations")}</div>
              )}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

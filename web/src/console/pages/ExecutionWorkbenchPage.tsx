import { useState, type ReactNode } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import { useCommand } from "../useCommand";
import { usePrincipal } from "../usePrincipal";
import { useI18n } from "../../i18n/I18nProvider";
import { PageHeader } from "../layout/PageHeader";
import { PageTrailProvider } from "../layout/PageTrail";
import { ConfirmDialog } from "../feedback/ConfirmDialog";
import { StatusBadge } from "../ui/StatusBadge";
import { RelativeTime } from "../ui/RelativeTime";
import { EmptyState } from "../ui/EmptyState";
import { ErrorState } from "../ui/ErrorState";
import { Skeleton } from "../ui/Skeleton";
import { StepRail } from "../components/StepRail";
import { MeasurementForm } from "../components/MeasurementForm";
import { stepStateTone } from "../labels";
import type { Execution, Step } from "../../types";

/**
 * ExecutionWorkbenchPage: the pilot anchor page. LOTO step rail with
 * confirmation on safety-significant steps, validated measurements, readable
 * observations, evidence, and report drafting with toast feedback.
 */
export function ExecutionWorkbenchPage() {
  const { executionId } = useParams<{ executionId: string }>();
  const { t, locale } = useI18n();
  const navigate = useNavigate();
  const { data: principal } = usePrincipal();
  const execution = useQuery(() => api.execution(executionId ?? ""));
  const incident = useQuery(
    () =>
      execution.data?.incident_id
        ? api.incident(execution.data.incident_id)
        : Promise.resolve(undefined),
    [execution.data?.incident_id],
  );
  const [pendingStep, setPendingStep] = useState<Step | null>(null);

  const canWrite = principal?.permissions.includes("execution:write") ?? false;

  const start = useCommand(
    (value: Execution) => api.startExecution(value),
    {
      successMessage: t("workbench.startSuccess"),
      onSuccess: () => void execution.refetch(),
    },
  );

  const completeStep = useCommand(
    (value: Execution, step: Step) => api.completeStep(value, step),
    {
      successMessage: t("workbench.stepComplete"),
      onSuccess: () => {
        setPendingStep(null);
        void execution.refetch();
      },
    },
  );

  const draftReport = useCommand(
    (id: string) => api.draftReport(id),
    {
      successMessage: t("workbench.reportDrafted"),
      onSuccess: (report) => navigate(`/reports/${report.id}`),
    },
  );

  const recordMeasurement = useCommand(
    (value: Execution, type: string, measure: string, unit: string) =>
      api.recordMeasurement(value, type, measure, unit),
    {
      onSuccess: () => void execution.refetch(),
    },
  );

  if (execution.error) {
    return (
      <section>
        <PageHeader title={t("nav.executions")} principal={principal} />
        <ErrorState
          message={execution.error}
          onRetry={() => void execution.refetch()}
          onBack={() => navigate("/executions")}
        />
      </section>
    );
  }
  if (execution.loading || !execution.data) {
    return (
      <section>
        <PageHeader title={t("nav.executions")} principal={principal} />
        <Skeleton height={300} />
      </section>
    );
  }

  const value = execution.data;
  const handleStepToggle = (step: Step) => {
    if (step.risk_level !== "INFORMATIONAL") {
      setPendingStep(step);
      return;
    }
    void completeStep.run(value, step);
  };

  const trail = [
    { label: t("nav.incidents"), to: "/incidents" },
    ...(incident.data
      ? [{ label: incident.data.number, to: `/incidents/${incident.data.id}` }]
      : []),
    { label: t("nav.executions") },
  ];

  return (
    <PageTrailProvider trail={trail}>
      <section>
        <PageHeader
          title={value.purpose}
          principal={principal}
          actions={
            <>
              {value.state === "ASSIGNED" && canWrite && (
                <button
                  className="primary-button"
                  disabled={start.pending}
                  onClick={() => void start.run(value)}
                >
                  {t("execution.startInspection")}
                </button>
              )}
              {canWrite && value.state !== "COMPLETED" && (
                <button
                  className="secondary-button"
                  disabled={draftReport.pending}
                  onClick={() => void draftReport.run(value.id)}
                >
                  {t("execution.draftReport")}
                </button>
              )}
            </>
          }
        />
        <div className="workbench-meta">
          <StatusBadge
            tone={stepStateTone(value.state)}
            label={value.state.replace("_", " ")}
          />
          {value.asset_tag ? (
            <Link to={`/assets/${value.asset_id}`} className="mono">
              {value.asset_tag}
            </Link>
          ) : null}
        </div>
        <div className="workbench-layout">
          <div className="workbench-steps">
            {value.steps.some((step) => step.state === "BLOCKED") && (
              <div className="loto-banner" role="status">
                <span className="loto-banner-dot" aria-hidden="true" />
                <div>
                  <strong>{t("workbench.lotoActive")}</strong>
                  <p>{t("workbench.lotoBody")}</p>
                </div>
              </div>
            )}
            <StepRail steps={value.steps} onToggle={handleStepToggle} canWrite={canWrite} />
          </div>
          <div className="workbench-rail" style={{ display: "grid", gap: 16 }}>
            <div className="panel">
              <div className="panel-heading">
                <h2>{t("execution.measurements")}</h2>
                <span className="count">{value.measurements.length}</span>
              </div>
              <div style={{ padding: 16 }}>
                {canWrite && (
                  <MeasurementForm
                    pending={recordMeasurement.pending}
                    onSubmit={(type, measure, unit) =>
                      void recordMeasurement.run(value, type, measure, unit)
                    }
                  />
                )}
                <div className="measurement-list" aria-live="polite">
                  {value.measurements.map((measurement) => (
                    <div key={measurement.id} className="measurement-row">
                      <span className="mono strong">
                        {measurement.value} {measurement.unit}
                      </span>
                      <span>{measurement.measurement_type}</span>
                      <RelativeTime time={measurement.observed_at} locale={locale} />
                    </div>
                  ))}
                  {value.measurements.length === 0 && (
                    <EmptyState title={t("execution.noMeasurements")} />
                  )}
                </div>
              </div>
            </div>
            <div className="panel">
              <div className="panel-heading">
                <h2>{t("execution.observations")}</h2>
                <span className="count">{value.observations.length}</span>
              </div>
              <div className="readable-list">
                {value.observations.length > 0 ? (
                  value.observations.map((observation, index) => (
                    <div key={index} className="readable-row">
                      <ReadableUnknown value={observation} />
                    </div>
                  ))
                ) : (
                  <EmptyState title={t("execution.noObservations")} />
                )}
              </div>
            </div>
            <div className="panel">
              <div className="panel-heading">
                <h2>{t("workbench.evidence")}</h2>
                <span className="count">{value.actions.length}</span>
              </div>
              <div className="readable-list">
                {value.actions.length > 0 ? (
                  value.actions.map((action, index) => (
                    <div key={index} className="readable-row">
                      <ReadableUnknown value={action} />
                    </div>
                  ))
                ) : (
                  <EmptyState title={t("workbench.noEvidence")} />
                )}
              </div>
            </div>
          </div>
        </div>
        <ConfirmDialog
          open={pendingStep !== null}
          onOpenChange={(open) => {
            if (!open) setPendingStep(null);
          }}
          title={t("workbench.confirmStep")}
          message={t("workbench.confirmStepBody")}
          confirmLabel={t("execution.complete")}
          pending={completeStep.pending}
          onConfirm={() => {
            if (pendingStep) void completeStep.run(value, pendingStep);
          }}
        />
      </section>
    </PageTrailProvider>
  );
}

/** ReadableUnknown: renders unknown payloads as key: value rows instead of a
 * raw JSON dump (observations/actions are untyped from the API today). */
function ReadableUnknown({ value }: { value: unknown }) {
  if (value === null || value === undefined) return <span className="muted">—</span>;
  if (typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>);
    if (entries.length === 0) return <span className="muted">—</span>;
    return (
      <div className="readable-object">
        {entries.map(([key, val]) => (
          <div key={key} className="readable-key-value">
            <span className="mono muted">{key}</span>
            <span>{renderLeaf(val)}</span>
          </div>
        ))}
      </div>
    );
  }
  return <span>{String(value)}</span>;
}

function renderLeaf(val: unknown): ReactNode {
  if (val === null || val === undefined) return "—";
  if (typeof val === "object") return JSON.stringify(val);
  return String(val);
}

import { useMemo, useState, type FormEvent } from "react";
import { CheckCircle, Sparkle } from "@phosphor-icons/react";
import { useI18n } from "../../i18n/I18nProvider";
import { FormField } from "../ui/FormField";
import { priorityLabelKey } from "../labels";
import { structureDescription } from "./structureDescription";
import { CustomFieldControl } from "./CustomFieldControl";
import type { Asset, CustomFieldDefinition, Person, Principal, Team } from "../../types";

const PRIORITIES = ["LOW", "MEDIUM", "HIGH", "CRITICAL"] as const;
const STATUSES = ["OPEN", "IN_PROGRESS", "RESOLVED", "CLOSED", "REOPENED"] as const;
const ACCEPTED_MEDIA = "image/*,video/*";

export interface CreateIncidentValue {
  site_id: string;
  asset_id: string;
  summary: string;
  details?: string;
  priority: string;
  status: string;
  assignee_id?: string;
  reporter_id: string;
  team_id?: string;
  occurred_at?: string;
  files: File[];
  custom_values?: Record<string, unknown>;
}

/**
 * CreateIncidentForm: Jira-style ticket creation. The left column leads
 * with a natural-language "describe what happened" box; the Structure
 * action paraphrases it into a summary, priority, asset and structured
 * details. The right rail carries the issue fields.
 */
export function CreateIncidentForm(props: {
  assets: Asset[];
  people: Person[];
  teams: Team[];
  principal: Principal | undefined;
  customFields: CustomFieldDefinition[];
  pending: boolean;
  onCancel: () => void;
  onCreate: (value: CreateIncidentValue) => void;
}) {
  const { t } = useI18n();
  const sites = useMemo(
    () => [...new Set(props.assets.map((asset) => asset.site_id))],
    [props.assets],
  );
  const today = useMemo(() => new Date().toISOString().slice(0, 10), []);
  const [siteID, setSiteID] = useState(sites[0] ?? "");
  const [assetID, setAssetID] = useState("");
  const [status, setStatus] = useState("OPEN");
  const [files, setFiles] = useState<File[]>([]);
  const [describe, setDescribe] = useState("");
  const [structured, setStructured] = useState(false);
  const [summary, setSummary] = useState("");
  const [details, setDetails] = useState("");
  const [assigneeID, setAssigneeID] = useState("");
  const [priority, setPriority] = useState("");
  const [occurredAt, setOccurredAt] = useState(today);
  const [teamID, setTeamID] = useState("");
  const [reporterID, setReporterID] = useState(props.principal?.id ?? "");
  const [customValues, setCustomValues] = useState<Record<string, unknown>>({});
  const [touched, setTouched] = useState(false);

  const siteAssets = useMemo(
    () => props.assets.filter((asset) => asset.site_id === siteID),
    [props.assets, siteID],
  );
  const asset = siteAssets.find((item) => item.id === assetID);
  const errors = {
    site: !siteID ? t("form.required") : undefined,
    asset: !assetID ? t("form.required") : undefined,
    summary: summary.trim().length < 3 ? t("form.required") : undefined,
    priority: !priority ? t("form.required") : undefined,
  };
  const customFieldErrors = props.customFields.reduce<Record<string, string | undefined>>(
    (acc, field) => {
      const value = customValues[field.key];
      const empty =
        value === undefined || value === null || value === "" ||
        (Array.isArray(value) && value.length === 0);
      if (field.config.required && empty) acc[field.key] = t("form.required");
      return acc;
    },
    {},
  );
  const customValid = Object.values(customFieldErrors).every((error) => error === undefined);
  const valid = !errors.site && !errors.asset && !errors.summary && !errors.priority && customValid;

  const structureLabels = {
    what: t("incident.structure.what"),
    observations: t("incident.structure.observations"),
    impact: t("incident.structure.impact"),
    actions: t("incident.structure.actions"),
  };

  function applyStructure() {
    if (!describe.trim()) return;
    const parsed = structureDescription(
      describe,
      siteAssets.map((item) => ({ id: item.id, tag: item.tag })),
      structureLabels,
    );
    setSummary(parsed.summary);
    setDetails(parsed.details);
    if (parsed.priority) setPriority(parsed.priority);
    if (parsed.assetId) {
      setAssetID(parsed.assetId);
      const matched = siteAssets.find((item) => item.id === parsed.assetId);
      if (matched) setSiteID(matched.site_id);
    }
    setStructured(true);
  }

  function submit(event: FormEvent) {
    event.preventDefault();
    setTouched(true);
    if (!valid || !asset) return;
    props.onCreate({
      site_id: asset.site_id,
      asset_id: asset.id,
      summary: summary.trim(),
      details: details.trim() || undefined,
      priority,
      status,
      assignee_id: assigneeID || undefined,
      reporter_id: reporterID,
      team_id: teamID || undefined,
      occurred_at: occurredAt ? new Date(`${occurredAt}T00:00:00`).toISOString() : undefined,
      files,
      custom_values: Object.keys(customValues).length > 0 ? customValues : undefined,
    });
  }

  return (
    <form className="create-incident-form" onSubmit={submit}>
      <div className="create-incident-main">
        <div className="describe-box">
          <FormField
            label={t("form.describe")}
            htmlFor="incident-describe"
            hint={t("form.structureHint")}
          >
            <textarea
              id="incident-describe"
              rows={4}
              placeholder={t("form.describePrompt")}
              value={describe}
              onChange={(event) => {
                setDescribe(event.target.value);
                setStructured(false);
              }}
            />
          </FormField>
          <div className="describe-actions">
            <button
              type="button"
              className="secondary-button"
              onClick={applyStructure}
              disabled={!describe.trim() || props.pending}
            >
              <Sparkle size={13} aria-hidden="true" />
              {t("form.structure")}
            </button>
            {structured ? (
              <span className="describe-feedback" role="status">
                <CheckCircle size={13} aria-hidden="true" />
                {t("form.structureDone")}
              </span>
            ) : null}
          </div>
        </div>
        <FormField label={t("form.summary")} htmlFor="incident-summary" required error={touched ? errors.summary : undefined}>
          <input
            id="incident-summary"
            value={summary}
            onChange={(event) => setSummary(event.target.value)}
            required
          />
        </FormField>
        <FormField label={t("form.details")} htmlFor="incident-details">
          <textarea
            id="incident-details"
            rows={5}
            placeholder={t("form.detailsPrompt")}
            value={details}
            onChange={(event) => setDetails(event.target.value)}
          />
        </FormField>
        <FormField label={t("form.media")} htmlFor="incident-media" hint={t("incident.media.hint")}>
          <input
            id="incident-media"
            type="file"
            accept={ACCEPTED_MEDIA}
            multiple
            onChange={(event) => setFiles(Array.from(event.target.files ?? []))}
          />
          {files.length > 0 ? (
            <p className="form-hint">
              {files.map((file) => file.name).join(", ")}
            </p>
          ) : null}
        </FormField>
      </div>

      <div className="create-incident-rail">
        <div className="dialog-context-row">
          <span className="eyebrow">{t("site.switcherLabel")}</span>
          <select
            aria-label={t("site.switcherLabel")}
            value={siteID}
            onChange={(event) => {
              setSiteID(event.target.value);
              setAssetID("");
            }}
          >
            {sites.length === 0 ? (
              <option value="">{t("form.selectSite")}</option>
            ) : (
              sites.map((id) => (
                <option key={id} value={id}>
                  {id}
                </option>
              ))
            )}
          </select>
        </div>
        <FormField label={t("form.asset")} htmlFor="incident-asset" required error={touched ? errors.asset : undefined}>
          <select
            id="incident-asset"
            value={assetID}
            onChange={(event) => setAssetID(event.target.value)}
            required
          >
            <option value="" disabled>
              {siteAssets.length === 0
                ? t("form.selectAssetForSite")
                : t("form.selectAsset")}
            </option>
            {siteAssets.map((item) => (
              <option key={item.id} value={item.id}>
                {item.tag} · {item.name}
              </option>
            ))}
          </select>
        </FormField>
        <FormField label={t("form.status")} htmlFor="incident-status">
          <select id="incident-status" value={status} onChange={(event) => setStatus(event.target.value)}>
            {STATUSES.map((value) => (
              <option key={value} value={value}>
                {t(`incident.status.${value.toLowerCase()}` as never)}
              </option>
            ))}
          </select>
        </FormField>
        <FormField label={t("form.priority")} htmlFor="incident-priority" required error={touched ? errors.priority : undefined}>
          <select
            id="incident-priority"
            value={priority}
            onChange={(event) => setPriority(event.target.value)}
            required
          >
            <option value="" disabled>
              {t("incidents.tabs.all")}
            </option>
            {PRIORITIES.map((level) => (
              <option key={level} value={level}>
                {t(priorityLabelKey(level))}
              </option>
            ))}
          </select>
        </FormField>
        <FormField label={t("form.assignee")} htmlFor="incident-assignee">
          <select id="incident-assignee" value={assigneeID} onChange={(event) => setAssigneeID(event.target.value)}>
            <option value="">{t("form.selectPerson")}</option>
            {props.people.map((person) => (
              <option key={person.id} value={person.id}>
                {person.display_name}
              </option>
            ))}
          </select>
        </FormField>
        <FormField label={t("form.reporter")} htmlFor="incident-reporter">
          <select id="incident-reporter" value={reporterID} onChange={(event) => setReporterID(event.target.value)}>
            {props.people.map((person) => (
              <option key={person.id} value={person.id}>
                {person.display_name}
              </option>
            ))}
          </select>
        </FormField>
        <FormField label={t("form.team")} htmlFor="incident-team">
          <select id="incident-team" value={teamID} onChange={(event) => setTeamID(event.target.value)}>
            <option value="">{t("form.selectTeam")}</option>
            {props.teams.map((team) => (
              <option key={team.id} value={team.id}>
                {team.name}
              </option>
            ))}
          </select>
        </FormField>
        <FormField label={t("form.date")} htmlFor="incident-date">
          <input
            id="incident-date"
            type="date"
            value={occurredAt}
            onChange={(event) => setOccurredAt(event.target.value)}
          />
        </FormField>
      </div>

      {props.customFields.length > 0 && (
        <fieldset className="form-section">
          <legend>{t("incident.customFields")}</legend>
          {props.customFields.map((field) => (
            <CustomFieldControl
              key={field.id}
              field={field}
              value={customValues[field.key]}
              onChange={(value) =>
                setCustomValues((prev) => ({ ...prev, [field.key]: value }))
              }
            />
          ))}
        </fieldset>
      )}

      <div className="dialog-footer">
        <button type="button" className="secondary-button" onClick={props.onCancel} disabled={props.pending}>
          {t("form.cancel")}
        </button>
        <button type="submit" className="primary-button" disabled={props.pending}>
          {t("form.create")}
        </button>
      </div>
    </form>
  );
}

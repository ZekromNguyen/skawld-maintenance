import { useRef, useState } from "react";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { errorMessage, useQuery } from "../useQuery";
import { useMutation } from "../useMutation";
import { ErrorState } from "../ui/ErrorState";
import { EmptyState } from "../ui/EmptyState";
import { PageHeader } from "../layout/PageHeader";
import { ConfirmDialog } from "../feedback/ConfirmDialog";
import { Dialog } from "../feedback/Dialog";
import { CreateFieldDialog } from "../components/CreateFieldDialog";
import { formatCustomValue } from "../components/customFieldFormat";
import type { CustomFieldDefinition } from "../../types";

/**
 * CustomFieldsPage: tenant-admin management of incident field definitions.
 * Create, edit (label/description/order only once data exists), retire
 * (keep history), and inspect value-change history.
 */
export function CustomFieldsPage() {
  const { t } = useI18n();
  const fields = useQuery(() => api.listFieldDefinitions("incident").then((list) => list.items), []);
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<CustomFieldDefinition | undefined>(undefined);
  const [retiring, setRetiring] = useState<CustomFieldDefinition | undefined>(undefined);
  const [history, setHistory] = useState<CustomFieldDefinition | undefined>(undefined);
  const [historyEntries, setHistoryEntries] = useState<
    { incident_id: string; value_before?: unknown; value_after: unknown; changed_at: string }[] | undefined
  >(undefined);
  const [dialogError, setDialogError] = useState<string | undefined>(undefined);

  const saveErrorRef = useRef<string | undefined>(undefined);
  const save = useMutation(async (value: Parameters<typeof api.createFieldDefinition>[0], id?: string) => {
    saveErrorRef.current = undefined;
    try {
      if (id) {
        return await api.updateFieldDefinition(id, {
          label: value.label,
          description: value.description,
          field_type: value.field_type,
          config: value.config,
          sort_order: value.sort_order,
          expected_version: editing?.version ?? 1,
        });
      }
      return await api.createFieldDefinition(value);
    } catch (err) {
      saveErrorRef.current = errorMessage(err);
      return undefined;
    }
  });
  const retire = useMutation((id: string) => api.retireFieldDefinition(id));
  const openHistory = useMutation((id: string) =>
    api.fieldDefinitionHistory(id).then((response) => response.items),
  );

  async function handleSave(value: Parameters<typeof api.createFieldDefinition>[0]) {
    setDialogError(undefined);
    const result = await save.run(value, editing?.id);
    if (result === undefined) {
      setDialogError(saveErrorRef.current ?? t("admin.customFields.saveFailed"));
      return;
    }
    setCreating(false);
    setEditing(undefined);
    void fields.refetch();
  }

  const [retireError, setRetireError] = useState<string | undefined>(undefined);
  async function handleRetire() {
    if (!retiring) return;
    setRetireError(undefined);
    const result = await retire.run(retiring.id);
    if (result === undefined) {
      setRetireError(retire.error ?? t("admin.customFields.retireFailed"));
      return;
    }
    setRetiring(undefined);
    void fields.refetch();
  }

  async function handleOpenHistory(field: CustomFieldDefinition) {
    setHistory(field);
    setHistoryEntries(undefined);
    const items = await openHistory.run(field.id);
    if (items === undefined) {
      setHistoryEntries(undefined);
      return;
    }
    setHistoryEntries(items);
  }

  if (fields.loading) return <div className="page-loading" aria-busy="true" />;
  if (fields.error) return <ErrorState message={fields.error} onRetry={() => void fields.refetch()} />;

  return (
    <div className="page">
      <PageHeader
        title={t("admin.customFields.title")}
        actions={
          <button className="primary-button" onClick={() => { setDialogError(undefined); setCreating(true); }}>
            {t("admin.customFields.newField")}
          </button>
        }
      />
      <section className="data-card">
        {fields.data && fields.data.length === 0 ? (
          <EmptyState title={t("admin.customFields.empty")} />
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("admin.customFields.label")}</th>
                <th>{t("admin.customFields.key")}</th>
                <th>{t("admin.customFields.type")}</th>
                <th>{t("admin.customFields.required")}</th>
                <th>{t("admin.customFields.sortOrder")}</th>
                <th>{t("admin.customFields.status")}</th>
                <th aria-label={t("admin.customFields.actions")} />
              </tr>
            </thead>
            <tbody>
              {fields.data?.map((field) => (
                <tr key={field.id}>
                  <td>{field.label}</td>
                  <td className="mono">{field.key}</td>
                  <td>{field.field_type}</td>
                  <td>{field.config.required ? "✓" : ""}</td>
                  <td>{field.sort_order}</td>
                  <td>{field.status === "ACTIVE" ? t("admin.customFields.active") : t("admin.customFields.retired")}</td>
                  <td className="row-actions">
                    <button className="secondary-button" onClick={() => { setDialogError(undefined); setEditing(field); }}>
                      {t("admin.customFields.edit")}
                    </button>
                    <button className="secondary-button" onClick={() => handleOpenHistory(field)}>
                      {t("admin.customFields.history")}
                    </button>
                    <button
                      className="secondary-button danger"
                      disabled={field.status === "RETIRED"}
                      onClick={() => setRetiring(field)}
                    >
                      {t("admin.customFields.retire")}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {(creating || editing) && (
        <CreateFieldDialog
          field={editing}
          pending={save.pending}
          error={dialogError}
          onCreate={handleSave}
          onClose={() => {
            setCreating(false);
            setEditing(undefined);
          }}
        />
      )}
      {retiring && (
        <ConfirmDialog
          open
          onOpenChange={() => setRetiring(undefined)}
          title={t("admin.customFields.retireConfirmTitle")}
          message={retireError
            ? `${t("admin.customFields.retireConfirmCount", {
                field: retiring.label,
                count: retiring.incident_count ?? 0,
              })} ${retireError}`
            : t("admin.customFields.retireConfirmCount", {
                field: retiring.label,
                count: retiring.incident_count ?? 0,
              })}
          confirmLabel={t("admin.customFields.retire")}
          pending={retire.pending}
          onConfirm={() => void handleRetire()}
        />
      )}
      {history && (
        <Dialog
          open
          onOpenChange={() => setHistory(undefined)}
          title={`${t("admin.customFields.history")}: ${history.label}`}
          footer={<button className="secondary-button" onClick={() => setHistory(undefined)}>{t("admin.customFields.close")}</button>}
        >
          {openHistory.error ? (
            <p className="form-error" role="alert">{openHistory.error}</p>
          ) : historyEntries === undefined ? (
            <div className="page-loading" aria-busy="true" />
          ) : historyEntries.length === 0 ? (
            <p>{t("admin.customFields.historyEmpty")}</p>
          ) : (
            <ul className="history-list">
              {historyEntries.map((entry, index) => (
                <li key={index} className="history-item">
                  <span className="mono">{formatCustomValue(history, entry.value_after)}</span>
                  <time>{new Date(entry.changed_at).toLocaleString()}</time>
                </li>
              ))}
            </ul>
          )}
        </Dialog>
      )}
    </div>
  );
}

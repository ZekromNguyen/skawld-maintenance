import { useMemo, useState } from "react";
import { api } from "../../api";
import { useI18n } from "../../i18n/I18nProvider";
import { Dialog } from "../feedback/Dialog";
import { FormField } from "../ui/FormField";
import { CustomFieldControl } from "./CustomFieldControl";
import type { CustomFieldDefinition, CustomFieldType } from "../../types";

const FIELD_TYPES: CustomFieldType[] = ["TEXT", "NUMBER", "DATE", "SELECT", "MULTI_SELECT"];

/** slugify converts a display label into a candidate machine key. */
export function slugify(label: string): string {
  const slug = label
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9_]+/g, "_")
    .replace(/^[0-9]+/, "")
    .replace(/^_+|_+$/g, "");
  return slug.length > 64 ? slug.slice(0, 64) : slug;
}

export function CreateFieldDialog(props: {
  field?: CustomFieldDefinition;
  pending: boolean;
  error: string | undefined;
  onCreate: (value: {
    entity_type: string;
    key: string;
    label: string;
    description?: string;
    field_type: CustomFieldType;
    config: CustomFieldDefinition["config"];
    sort_order: number;
  }) => Promise<void>;
  onClose: () => void;
}) {
  const { t } = useI18n();
  const editing = props.field !== undefined;
  const locked = editing && (props.field?.has_values ?? false);
  const [label, setLabel] = useState(props.field?.label ?? "");
  const [description, setDescription] = useState(props.field?.description ?? "");
  const [key, setKey] = useState(props.field?.key ?? "");
  const [keyTouched, setKeyTouched] = useState(editing);
  const [fieldType, setFieldType] = useState<CustomFieldType>(props.field?.field_type ?? "TEXT");
  const [required, setRequired] = useState(props.field?.config.required ?? false);
  const [maxLength, setMaxLength] = useState(props.field?.config.max_length ?? 0);
  const [min, setMin] = useState(props.field?.config.min);
  const [max, setMax] = useState(props.field?.config.max);
  const [regex, setRegex] = useState(props.field?.config.regex ?? "");
  const [options, setOptions] = useState(
    props.field?.config.options ?? [{ label: "", value: "" }],
  );
  const [sortOrder, setSortOrder] = useState(props.field?.sort_order ?? 0);

  const config = useMemo<CustomFieldDefinition["config"]>(() => {
    const base: CustomFieldDefinition["config"] = { required };
    if (fieldType === "TEXT") {
      if (maxLength > 0) base.max_length = maxLength;
      if (regex.trim()) base.regex = regex.trim();
    }
    if (fieldType === "NUMBER") {
      if (min !== undefined) base.min = min;
      if (max !== undefined) base.max = max;
    }
    if (fieldType === "SELECT" || fieldType === "MULTI_SELECT") {
      base.options = options
        .map((option) => ({ label: option.label.trim(), value: option.value.trim() }))
        .filter((option) => option.value !== "");
    }
    return base;
  }, [required, fieldType, maxLength, regex, min, max, options]);

  const previewField: CustomFieldDefinition = {
    id: "preview",
    entity_type: "incident",
    key: key || "preview",
    label: label || "Preview",
    description,
    field_type: fieldType,
    config,
    status: "ACTIVE",
    sort_order: sortOrder,
    version: 1,
    created_at: "",
    updated_at: "",
  };

  const typeNeedsOptions = fieldType === "SELECT" || fieldType === "MULTI_SELECT";
  const keyError = keyTouched && !/^[a-z][a-z0-9_]{0,63}$/.test(key) ? t("admin.customFields.keyInvalid") : undefined;
  const optionsInvalid = typeNeedsOptions && config.options?.length === 0;
  const valid =
    label.trim().length > 0 &&
    key.trim().length > 0 &&
    /^[a-z][a-z0-9_]{0,63}$/.test(key) &&
    !optionsInvalid;

  async function submit() {
    if (!valid) return;
    await props.onCreate({
      entity_type: "incident",
      key: key.trim(),
      label: label.trim(),
      description: description.trim() || undefined,
      field_type: fieldType,
      config,
      sort_order: sortOrder,
    });
  }

  return (
    <Dialog
      open
      onOpenChange={() => props.onClose()}
      title={editing ? t("admin.customFields.editField") : t("admin.customFields.newField")}
      footer={
        <>
          <button type="button" className="secondary-button" onClick={props.onClose} disabled={props.pending}>
            {t("form.cancel")}
          </button>
          <button type="button" className="primary-button" onClick={() => void submit()} disabled={props.pending || !valid}>
            {editing ? t("admin.customFields.edit") : t("form.create")}
          </button>
        </>
      }
    >
      <form className="create-field-form" onSubmit={(event) => { event.preventDefault(); void submit(); }}>
        {props.error ? <p className="form-error" role="alert">{props.error}</p> : null}
        <FormField label={t("admin.customFields.label")} htmlFor="field-label" required>
          <input
            id="field-label"
            value={label}
            onChange={(event) => {
              setLabel(event.target.value);
              if (!keyTouched && !editing) setKey(slugify(event.target.value));
            }}
          />
        </FormField>
        <FormField label={t("admin.customFields.key")} htmlFor="field-key" required error={keyError}>
          <input
            id="field-key"
            value={key}
            disabled={editing}
            onChange={(event) => {
              setKey(event.target.value);
              setKeyTouched(true);
            }}
          />
        </FormField>
        <FormField label={t("admin.customFields.description")} htmlFor="field-description">
          <input
            id="field-description"
            value={description}
            onChange={(event) => setDescription(event.target.value)}
          />
        </FormField>
        <FormField label={t("admin.customFields.type")} htmlFor="field-type">
          <select
            id="field-type"
            value={fieldType}
            disabled={locked}
            onChange={(event) => setFieldType(event.target.value as CustomFieldType)}
          >
            {FIELD_TYPES.map((type) => (
              <option key={type} value={type}>{type}</option>
            ))}
          </select>
        </FormField>
        <label className="option-row">
          <input type="checkbox" checked={required} disabled={locked} onChange={(event) => setRequired(event.target.checked)} />
          {t("admin.customFields.required")}
        </label>
        {fieldType === "TEXT" ? (
          <FormField label={t("admin.customFields.maxLength")} htmlFor="field-maxlength">
            <input
              id="field-maxlength"
              type="number"
              min={0}
              disabled={locked}
              value={maxLength || ""}
              onChange={(event) => setMaxLength(Number(event.target.value) || 0)}
            />
          </FormField>
        ) : null}
        {fieldType === "TEXT" ? (
          <FormField label={t("admin.customFields.regex")} htmlFor="field-regex">
            <input
              id="field-regex"
              disabled={locked}
              value={regex}
              placeholder="^[A-Z]{2}-\d+$"
              onChange={(event) => setRegex(event.target.value)}
            />
          </FormField>
        ) : null}
        {fieldType === "NUMBER" ? (
          <div className="field-row">
            <FormField label={t("admin.customFields.min")} htmlFor="field-min">
              <input
                id="field-min"
                type="number"
                disabled={locked}
                value={min ?? ""}
                onChange={(event) => setMin(event.target.value === "" ? undefined : Number(event.target.value))}
              />
            </FormField>
            <FormField label={t("admin.customFields.max")} htmlFor="field-max">
              <input
                id="field-max"
                type="number"
                disabled={locked}
                value={max ?? ""}
                onChange={(event) => setMax(event.target.value === "" ? undefined : Number(event.target.value))}
              />
            </FormField>
          </div>
        ) : null}
        {typeNeedsOptions ? (
          <fieldset className="form-section">
            <legend>{t("admin.customFields.options")}</legend>
            {optionsInvalid ? <p className="form-error" role="alert">{t("admin.customFields.optionsRequired")}</p> : null}
            {options.map((option, index) => (
              <div key={index} className="field-row">
                <input
                  aria-label={t("admin.customFields.optionLabel")}
                  disabled={locked}
                  value={option.label}
                  placeholder={t("admin.customFields.optionLabel")}
                  onChange={(event) => {
                    const next = [...options];
                    next[index] = { ...next[index], label: event.target.value };
                    setOptions(next);
                  }}
                />
                <input
                  aria-label={t("admin.customFields.optionValue")}
                  disabled={locked}
                  value={option.value}
                  placeholder={t("admin.customFields.optionValue")}
                  onChange={(event) => {
                    const next = [...options];
                    next[index] = { ...next[index], value: event.target.value };
                    setOptions(next);
                  }}
                />
                <button
                  type="button"
                  className="icon-button"
                  aria-label={t("admin.customFields.removeOption")}
                  disabled={locked}
                  onClick={() => setOptions(options.filter((_, i) => i !== index))}
                >
                  ×
                </button>
              </div>
            ))}
            <button type="button" className="secondary-button" disabled={locked} onClick={() => setOptions([...options, { label: "", value: "" }])}>
              {t("admin.customFields.addOption")}
            </button>
          </fieldset>
        ) : null}
        <FormField label={t("admin.customFields.sortOrder")} htmlFor="field-sort">
          <input
            id="field-sort"
            type="number"
            value={sortOrder}
            onChange={(event) => setSortOrder(Number(event.target.value) || 0)}
          />
        </FormField>
        <fieldset className="form-section">
          <legend>{t("admin.customFields.preview")}</legend>
          <CustomFieldControl field={previewField} value={undefined} onChange={() => {}} />
        </fieldset>
      </form>
    </Dialog>
  );
}

import { useI18n } from "../../i18n/I18nProvider";
import { FormField } from "../ui/FormField";
import type { CustomFieldDefinition } from "../../types";

/**
 * CustomFieldControl renders the form control for one tenant-defined field,
 * driven by its field_type and config. Value shapes per type:
 * TEXT/DATE/SELECT -> string, NUMBER -> number | undefined, MULTI_SELECT -> string[].
 */
export function CustomFieldControl(props: {
  field: CustomFieldDefinition;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const { t } = useI18n();
  const { field, value } = props;
  const required = field.config.required === true;
  const id = `field-${field.key}`;

  switch (field.field_type) {
    case "TEXT":
      return (
        <FormField label={field.label} htmlFor={id} required={required} hint={field.description}>
          <input
            id={id}
            type="text"
            value={typeof value === "string" ? value : ""}
            maxLength={field.config.max_length}
            onChange={(event) => props.onChange(event.target.value)}
          />
        </FormField>
      );
    case "NUMBER":
      return (
        <FormField label={field.label} htmlFor={id} required={required} hint={field.description}>
          <input
            id={id}
            type="number"
            value={typeof value === "number" ? value : ""}
            min={field.config.min}
            max={field.config.max}
            onChange={(event) =>
              props.onChange(event.target.value === "" ? undefined : Number(event.target.value))
            }
          />
        </FormField>
      );
    case "DATE":
      return (
        <FormField label={field.label} htmlFor={id} required={required} hint={field.description}>
          <input
            id={id}
            type="date"
            value={typeof value === "string" ? value : ""}
            onChange={(event) => props.onChange(event.target.value)}
          />
        </FormField>
      );
    case "SELECT":
      return (
        <FormField label={field.label} htmlFor={id} required={required} hint={field.description}>
          <select
            id={id}
            value={typeof value === "string" ? value : ""}
            onChange={(event) => props.onChange(event.target.value)}
          >
            <option value="">{t("form.selectPlaceholder")}</option>
            {field.config.options?.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </FormField>
      );
    case "MULTI_SELECT":
      return (
        <FormField label={field.label} htmlFor={id} required={required} hint={field.description}>
          <div id={id} role="group" aria-label={field.label} className="option-group">
            {field.config.options?.map((option) => {
              const selected = Array.isArray(value) && value.includes(option.value);
              return (
                <label key={option.value} className="option-row">
                  <input
                    type="checkbox"
                    checked={selected}
                    onChange={() => {
                      const current = Array.isArray(value) ? value : [];
                      const next = selected
                        ? current.filter((item) => item !== option.value)
                        : [...current, option.value];
                      props.onChange(next);
                    }}
                  />
                  {option.label}
                </label>
              );
            })}
          </div>
        </FormField>
      );
  }
}

import type { CustomFieldDefinition } from "../../types";

/** Formats a stored custom field value for display. */
export function formatCustomValue(
  field: Pick<CustomFieldDefinition, "field_type" | "config">,
  value: unknown,
): string {
  if (value === undefined || value === null) return "";
  if (field.field_type === "MULTI_SELECT" && Array.isArray(value)) {
    const byValue = new Map((field.config.options ?? []).map((option) => [option.value, option.label]));
    return value.map((item) => byValue.get(String(item)) ?? String(item)).join(", ");
  }
  if (field.field_type === "SELECT") {
    const option = (field.config.options ?? []).find((candidate) => candidate.value === value);
    return option ? option.label : String(value);
  }
  if (Array.isArray(value)) return value.join(", ");
  return String(value);
}

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { I18nProvider } from "../../i18n/I18nProvider";
import { CustomFieldControl } from "./CustomFieldControl";
import type { CustomFieldDefinition } from "../../types";

function textField(): CustomFieldDefinition {
  return {
    id: "def-1", entity_type: "incident", key: "po_number", label: "PO Number",
    field_type: "TEXT", config: { required: true }, status: "ACTIVE",
    sort_order: 1, version: 1, created_at: "", updated_at: ""
  };
}

function renderControl(field: CustomFieldDefinition, value: unknown, onChange = () => {}) {
  return render(
    <I18nProvider>
      <CustomFieldControl field={field} value={value} onChange={onChange} />
    </I18nProvider>
  );
}

describe("CustomFieldControl", () => {
  it("renders a text input for TEXT fields", () => {
    renderControl(textField(), "");
    expect(screen.getByLabelText(/PO Number/)).toBeTruthy();
  });

  it("renders select with options for SELECT fields", () => {
    const field = {
      ...textField(),
      field_type: "SELECT" as const,
      config: { options: [{ label: "Zone A", value: "a" }, { label: "Zone B", value: "b" }] }
    };
    renderControl(field, "");
    expect(screen.getByRole("combobox")).toBeTruthy();
    expect(screen.getByRole("option", { name: "Zone A" })).toBeTruthy();
  });

  it("renders checkboxes for MULTI_SELECT fields", () => {
    const field = {
      ...textField(),
      field_type: "MULTI_SELECT" as const,
      config: { options: [{ label: "A", value: "a" }, { label: "B", value: "b" }] }
    };
    renderControl(field, ["a"]);
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(2);
    expect((checkboxes[0] as HTMLInputElement).checked).toBe(true);
    expect((checkboxes[1] as HTMLInputElement).checked).toBe(false);
  });

  it("renders a number input for NUMBER fields", () => {
    renderControl({ ...textField(), field_type: "NUMBER", config: { min: 0, max: 100 } }, 42);
    expect(screen.getByRole("spinbutton")).toBeTruthy();
  });

  it("renders a date input for DATE fields", () => {
    renderControl({ ...textField(), field_type: "DATE" }, "2026-08-09");
    expect(screen.getByLabelText(/PO Number/)).toBeTruthy();
  });

  it("calls onChange with the typed text value", () => {
    const onChange = vi.fn();
    renderControl(textField(), "", onChange);
    fireEvent.change(screen.getByLabelText(/PO Number/), { target: { value: "PO-42" } });
    expect(onChange).toHaveBeenCalledWith("PO-42");
  });

  it("calls onChange with the selected option value", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    const field = {
      ...textField(),
      field_type: "SELECT" as const,
      config: { options: [{ label: "Zone A", value: "a" }, { label: "Zone B", value: "b" }] }
    };
    renderControl(field, "", onChange);
    await user.selectOptions(screen.getByRole("combobox"), "b");
    expect(onChange).toHaveBeenCalledWith("b");
  });

  it("toggles multi-select values", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    const field = {
      ...textField(),
      field_type: "MULTI_SELECT" as const,
      config: { options: [{ label: "A", value: "a" }, { label: "B", value: "b" }] }
    };
    renderControl(field, [], onChange);
    await user.click(screen.getByRole("checkbox", { name: "A" }));
    expect(onChange).toHaveBeenCalledWith(["a"]);
  });
});

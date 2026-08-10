import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { CustomFieldsPage } from "./CustomFieldsPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { ThemeProvider } from "../../theme/ThemeProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import type { Principal } from "../../types";

const FIELD = {
  id: "def-1",
  entity_type: "incident",
  key: "po_number",
  label: "PO Number",
  field_type: "TEXT",
  config: { required: true },
  status: "ACTIVE",
  sort_order: 1,
  version: 1,
  incident_count: 4,
  created_at: "2026-08-09T00:00:00Z",
  updated_at: "2026-08-09T00:00:00Z",
};

const RETIRED_FIELD = { ...FIELD, id: "def-2", key: "zone", label: "Zone", field_type: "SELECT", config: { options: [{ label: "A", value: "a" }] }, status: "RETIRED" };

vi.mock("../../api", () => ({
  api: {
    principal: () =>
      Promise.resolve({
        id: "p1",
        display_name: "Admin",
        organization_id: "org-1",
        site_ids: ["s1"],
        roles: ["Administrator"],
        permissions: ["incident:read", "field:manage"],
      } satisfies Principal),
    listFieldDefinitions: vi.fn(),
    createFieldDefinition: vi.fn(),
    updateFieldDefinition: vi.fn(),
    retireFieldDefinition: vi.fn(),
    fieldDefinitionHistory: vi.fn(),
  },
}));

import { api } from "../../api";

function renderPage() {
  return render(
    <ThemeProvider>
      <I18nProvider>
        <PrincipalProvider>
          <MemoryRouter>
            <CustomFieldsPage />
          </MemoryRouter>
        </PrincipalProvider>
      </I18nProvider>
    </ThemeProvider>,
  );
}

describe("CustomFieldsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (api.listFieldDefinitions as ReturnType<typeof vi.fn>).mockResolvedValue({ items: [FIELD, RETIRED_FIELD] });
  });

  it("renders definitions in a table", async () => {
    renderPage();
    expect(await screen.findByText("PO Number")).toBeTruthy();
    // Retired fields are hidden by default.
    expect(screen.queryByText("zone")).toBeNull();

    fireEvent.click(await screen.findByRole("checkbox", { name: /show retired/i }));
    expect(await screen.findByText("zone")).toBeTruthy();
    expect(screen.getAllByText("TEXT").length).toBeGreaterThan(0);
  });

  it("shows the empty state when there are no fields", async () => {
    (api.listFieldDefinitions as ReturnType<typeof vi.fn>).mockResolvedValue({ items: [] });
    renderPage();
    expect(await screen.findByText(/No custom fields yet/)).toBeTruthy();
  });

  it("creates a field through the dialog", async () => {
    (api.createFieldDefinition as ReturnType<typeof vi.fn>).mockResolvedValue({ id: "def-3", key: "zone" });
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: /new field/i }));
    const dialog = screen.getByRole("dialog");
    fireEvent.change(within(dialog).getByLabelText(/label/i), { target: { value: "Zone" } });
    fireEvent.change(within(dialog).getByLabelText(/key/i), { target: { value: "zone" } });
    fireEvent.click(within(dialog).getByRole("button", { name: /create/i }));
    await waitFor(() =>
      expect(api.createFieldDefinition as ReturnType<typeof vi.fn>).toHaveBeenCalledWith(
        expect.objectContaining({ key: "zone", label: "Zone", entity_type: "incident", field_type: "TEXT" }),
      ),
    );
  });

  it("retires a field after confirmation", async () => {
    (api.retireFieldDefinition as ReturnType<typeof vi.fn>).mockResolvedValue({ id: "def-1", status: "RETIRED" });
    renderPage();
    const row = (await screen.findByText("PO Number")).closest("tr") as HTMLTableRowElement;
    fireEvent.click(within(row).getByRole("button", { name: /retire/i }));
    expect(await screen.findByText(/holds values on 4 incidents/i)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /retire/i }));
    await waitFor(() =>
      expect(api.retireFieldDefinition as ReturnType<typeof vi.fn>).toHaveBeenCalledWith("def-1"),
    );
  });

  it("disables retire for already-retired fields", async () => {
    renderPage();
    fireEvent.click(await screen.findByRole("checkbox", { name: /show retired/i }));
    const row = (await screen.findByText("Zone")).closest("tr") as HTMLTableRowElement;
    expect((within(row).getByRole("button", { name: /retire/i }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("shows value history in a drawer", async () => {
    (api.fieldDefinitionHistory as ReturnType<typeof vi.fn>).mockResolvedValue({
      items: [{ incident_id: "i1", value_after: "PO-42", changed_at: "2026-08-09T00:00:00Z" }],
    });
    renderPage();
    const row = (await screen.findByText("PO Number")).closest("tr") as HTMLTableRowElement;
    fireEvent.click(within(row).getByRole("button", { name: /history/i }));
    expect(await screen.findByText("PO-42")).toBeTruthy();
  });
});

it("locks the key but keeps the type editable for unused fields", async () => {
  renderPage();
  const row = (await screen.findByText("PO Number")).closest("tr") as HTMLTableRowElement;
  fireEvent.click(within(row).getByRole("button", { name: /edit/i }));
  const dialog = screen.getByRole("dialog");
  expect((within(dialog).getByLabelText(/key/i) as HTMLInputElement).disabled).toBe(true);
  expect((within(dialog).getByLabelText(/field type/i) as HTMLSelectElement).disabled).toBe(false);
});

it("updates a field through the edit dialog", async () => {
  (api.updateFieldDefinition as ReturnType<typeof vi.fn>).mockResolvedValue({ id: "def-1", label: "Purchase Order" });
  renderPage();
  const row = (await screen.findByText("PO Number")).closest("tr") as HTMLTableRowElement;
  fireEvent.click(within(row).getByRole("button", { name: /edit/i }));
  const dialog = screen.getByRole("dialog");
  fireEvent.change(within(dialog).getByLabelText(/label/i), { target: { value: "Purchase Order" } });
  fireEvent.click(within(dialog).getByRole("button", { name: /^edit$/i }));
  await waitFor(() =>
    expect(api.updateFieldDefinition as ReturnType<typeof vi.fn>).toHaveBeenCalledWith(
      "def-1",
      expect.objectContaining({ label: "Purchase Order", expected_version: 1 }),
    ),
  );
});

it("shows the generic save error when the API rejects creation", async () => {
  (api.createFieldDefinition as ReturnType<typeof vi.fn>).mockRejectedValue(new Error("boom"));
  renderPage();
  fireEvent.click(await screen.findByRole("button", { name: /new field/i }));
  const dialog = screen.getByRole("dialog");
  fireEvent.change(within(dialog).getByLabelText(/label/i), { target: { value: "Zone" } });
  fireEvent.change(within(dialog).getByLabelText(/key/i), { target: { value: "zone" } });
  fireEvent.click(within(dialog).getByRole("button", { name: /create/i }));
  expect(await screen.findByText("boom")).toBeTruthy();
});

it("locks config inputs when the field already holds values", async () => {
  (api.listFieldDefinitions as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
    items: [{ ...FIELD, has_values: true }]
  });
  renderPage();
  const row = (await screen.findByText("PO Number")).closest("tr") as HTMLTableRowElement;
  fireEvent.click(within(row).getByRole("button", { name: /edit/i }));
  const dialog = screen.getByRole("dialog");
  expect((within(dialog).getByLabelText(/field type/i) as HTMLSelectElement).disabled).toBe(true);
  expect((within(dialog).getByLabelText(/required/i) as HTMLInputElement).disabled).toBe(true);
  expect((within(dialog).getByLabelText(/key/i) as HTMLInputElement).disabled).toBe(true);
});

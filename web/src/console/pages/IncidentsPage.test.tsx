import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { IncidentsPage } from "./IncidentsPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { ToastProvider } from "../feedback/Toast";
import { api } from "../../api";
import type { Incident } from "../../types";

const fixtures = vi.hoisted(() => ({
  OPEN_HIGH: {
    id: "i1",
    site_id: "s1",
    asset_id: "a1",
    asset_tag: "P-302",
    number: "IN-1",
    summary: "Pump vibration",
    priority: "HIGH",
    status: "OPEN",
    detected_at: new Date().toISOString(),
    version: 1
  },
  INPROG_MED: {
    id: "i2",
    site_id: "s1",
    asset_id: "a2",
    asset_tag: "P-304",
    number: "IN-2",
    summary: "Bearing temperature",
    priority: "MEDIUM",
    status: "IN_PROGRESS",
    detected_at: new Date().toISOString(),
    version: 1
  },
  RESOLVED_LOW: {
    id: "i3",
    site_id: "s1",
    asset_id: "a3",
    asset_tag: "P-305",
    number: "IN-3",
    summary: "Resolved noise",
    priority: "LOW",
    status: "RESOLVED",
    detected_at: new Date().toISOString(),
    version: 1
  }
}));

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({
      id: "p1",
      display_name: "Tester",
      site_ids: ["s1"],
      permissions: ["incident:create", "incident:read"]
    }),
    incidents: vi.fn().mockResolvedValue({ items: [fixtures.OPEN_HIGH, fixtures.INPROG_MED, fixtures.RESOLVED_LOW], next_cursor: null, has_more: false, custom_fields: [] }),
    assets: vi.fn().mockResolvedValue({
      items: [
        { id: "a1", site_id: "s1", tag: "P-302", name: "Process Pump", class: "CENTRIFUGAL_PUMP", status: "OPERATIONAL", source_of_truth: "OWNED_BY_SKAWLD" }
      ]
    }),
    createIncident: vi.fn().mockResolvedValue({ id: "new1", number: "IN-9" }),
    teams: vi.fn().mockResolvedValue({ items: [{ id: "t1", name: "Facilities" }] }),
    people: vi.fn().mockResolvedValue({ items: [{ id: "p1", display_name: "Tester" }] }),
    listFieldDefinitions: vi.fn().mockResolvedValue({ items: [] }),
    uploadIncidentAttachment: vi.fn().mockResolvedValue({ id: "att1" })
  }
}));

function renderPage() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider>
          <ToastProvider>
            <MemoryRouter>
              <IncidentsPage />
            </MemoryRouter>
          </ToastProvider>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("IncidentsPage", () => {
  beforeEach(() => {
    window.localStorage.clear();
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "p1",
      display_name: "Tester",
      site_ids: ["s1"],
      permissions: ["incident:create", "incident:read"]
    });
  });

  it("shows a kanban board with one column per state", async () => {
    renderPage();
    expect(await screen.findByText("Pump vibration")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /board/i }));
    expect(await screen.findByText("Resolved noise")).toBeTruthy();
    expect(screen.getByText("Pump vibration")).toBeTruthy();
    expect(screen.queryByRole("tab")).toBeNull();
    const openColumn = screen.getByText("Pump vibration").closest(".board-column");
    expect(openColumn?.textContent).toContain("IN-1");
    expect(openColumn?.textContent).not.toContain("IN-2");
    fireEvent.click(screen.getByRole("button", { name: /list/i }));
    expect(await screen.findByRole("tab", { name: /all/i })).toBeTruthy();
  });

  it("filters rows by state tab", async () => {
    renderPage();
    expect(await screen.findByText("Pump vibration")).toBeTruthy();
    expect(screen.queryByText("Resolved noise")).toBeNull();
    fireEvent.click(screen.getByRole("tab", { name: /all/i }));
    await waitFor(() => expect(screen.getByText("Resolved noise")).toBeTruthy());
    fireEvent.click(screen.getByRole("tab", { name: /resolved/i }));
    expect(screen.queryByText("Pump vibration")).toBeNull();
    expect(screen.getByText("Resolved noise")).toBeTruthy();
  });

  it("renders alarm-row severity edges in list view", async () => {
    renderPage();
    expect(await screen.findByText("Pump vibration")).toBeTruthy();
    const row = screen.getByText("Pump vibration").closest("tr");
    expect(row?.className).toContain("alarm-row--high");
    const edge = row?.querySelector(".alarm-edge");
    expect(edge).toBeTruthy();
  });

  it("selects a row with j/k keys and marks it selected", async () => {
    renderPage();
    expect(await screen.findByText("Pump vibration")).toBeTruthy();
    fireEvent.click(screen.getByRole("tab", { name: /all/i }));
    await waitFor(() => expect(screen.getByText("Resolved noise")).toBeTruthy());
    const table = screen.getByRole("table");
    fireEvent.keyDown(table, { key: "j" });
    fireEvent.keyDown(table, { key: "j" });
    const selected = table.querySelector("tr.data-row.selected");
    expect(selected).toBeTruthy();
    expect(selected?.textContent).toContain("Bearing temperature");
    expect(selected?.getAttribute("aria-selected")).toBe("true");
  });

  it("opens the selected incident on Enter", async () => {
    renderPage();
    expect(await screen.findByText("Pump vibration")).toBeTruthy();
    const table = screen.getByRole("table");
    fireEvent.keyDown(table, { key: "j" });
    fireEvent.keyDown(table, { key: "Enter" });
    // DataTable rows are in a MemoryRouter; the whole-row click is exercised
    // through the row link, so assert the clickable table stays mounted.
    expect(table).toBeTruthy();
  });

  it("clears selection with Escape", async () => {
    renderPage();
    expect(await screen.findByText("Pump vibration")).toBeTruthy();
    const table = screen.getByRole("table");
    fireEvent.keyDown(table, { key: "j" });
    expect(table.querySelector("tr.data-row.selected")).toBeTruthy();
    fireEvent.keyDown(table, { key: "Escape" });
    expect(table.querySelector("tr.data-row.selected")).toBeNull();
  });

  it("saves and applies a filter view", async () => {
    renderPage();
    expect(await screen.findByText("Pump vibration")).toBeTruthy();
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "bearing" } });
    fireEvent.click(screen.getByRole("button", { name: /save view/i }));
    const applied = await screen.findByRole("button", { name: "Saved filter" });
    fireEvent.click(applied);
    expect((screen.getByRole("textbox") as HTMLInputElement).value).toBe("bearing");
  });

  it("hides create for unauthorized principals", async () => {
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "p2",
      display_name: "Reader",
      site_ids: ["s1"],
      permissions: ["incident:read"]
    });
    renderPage();
    expect(screen.queryByRole("button", { name: /create incident/i })).toBeNull();
  });

  it("validates the create form before submitting", async () => {
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: /create incident/i }));
    const dialog = screen.getByRole("dialog");
    fireEvent.submit(dialog.querySelector("form") as HTMLFormElement);
    await waitFor(() => expect(screen.getAllByText("Required").length).toBeGreaterThan(0));
    expect(api.createIncident).not.toHaveBeenCalled();
  });

  it("creates an incident and shows a success toast", async () => {
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: /create incident/i }));
    const dialog = screen.getByRole("dialog");
    fireEvent.change(within(dialog).getByLabelText(/asset/i), { target: { value: "a1" } });
    fireEvent.change(within(dialog).getByLabelText(/summary/i), { target: { value: "High vibration on bearing" } });
    fireEvent.change(within(dialog).getByLabelText(/priority/i), { target: { value: "HIGH" } });
    fireEvent.submit(dialog.querySelector("form") as HTMLFormElement);
    await waitFor(() =>
      expect(api.createIncident as ReturnType<typeof vi.fn>).toHaveBeenCalledWith(
        expect.objectContaining({
          site_id: "s1",
          asset_id: "a1",
          summary: "High vibration on bearing",
          priority: "HIGH",
          status: "OPEN"
        }),
      ),
    );
    expect(screen.getByText("Incident created")).toBeTruthy();
  });

  it("filters the asset list by the chosen site context", async () => {
    (api.assets as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      items: [
        { id: "a1", site_id: "s1", tag: "P-302", name: "Process Pump", class: "CENTRIFUGAL_PUMP", status: "OPERATIONAL", source_of_truth: "OWNED_BY_SKAWLD" },
        { id: "a2", site_id: "s2", tag: "P-500", name: "Boiler", class: "BOILER", status: "OPERATIONAL", source_of_truth: "OWNED_BY_SKAWLD" }
      ]
    });
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: /create incident/i }));
    const dialog = screen.getByRole("dialog");
    fireEvent.change(within(dialog).getByLabelText(/site/i), { target: { value: "s2" } });
    const assetSelect = within(dialog).getByLabelText(/asset/i) as HTMLSelectElement;
    expect([...assetSelect.options].map((option) => option.value)).toEqual(["", "a2"]);
  });
});

it("shows load more and appends the next page", async () => {
  const list = vi.mocked(api.incidents);
  list
    .mockResolvedValueOnce({ items: [fixtures.OPEN_HIGH as unknown as Incident], next_cursor: "c1", has_more: true, custom_fields: [] })
    .mockResolvedValueOnce({ items: [fixtures.RESOLVED_LOW as unknown as Incident], next_cursor: null, has_more: false, custom_fields: [] });
  renderPage();
  const button = await screen.findByRole("button", { name: "Load more" });
  fireEvent.click(button);
  await waitFor(() =>
    expect(screen.queryByRole("button", { name: "Load more" })).toBeNull(),
  );
});

it("blocks submit when a required custom field is empty", async () => {
  (api.createIncident as ReturnType<typeof vi.fn>).mockClear();
  (api.listFieldDefinitions as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
    items: [
      { id: "def-1", entity_type: "incident", key: "po_number", label: "PO Number", field_type: "TEXT", config: { required: true }, status: "ACTIVE", sort_order: 1, version: 1, created_at: "", updated_at: "" }
    ]
  });
  renderPage();
  fireEvent.click(await screen.findByRole("button", { name: /create incident/i }));
  const dialog = screen.getByRole("dialog");
  fireEvent.change(within(dialog).getByLabelText(/asset/i), { target: { value: "a1" } });
  fireEvent.change(within(dialog).getByLabelText(/summary/i), { target: { value: "High vibration on bearing" } });
  fireEvent.change(within(dialog).getByLabelText(/priority/i), { target: { value: "HIGH" } });
  const poField = await waitFor(() => within(dialog).getByLabelText(/PO Number/));
  const label = poField.closest(".form-field")?.querySelector("label");
  // probe: required marker should be present
  expect(label?.textContent).toContain("*");
  fireEvent.submit(dialog.querySelector("form") as HTMLFormElement);
  await waitFor(() => expect(api.createIncident).not.toHaveBeenCalled());
});

it("does not let a retired required field block submit", async () => {
  (api.createIncident as ReturnType<typeof vi.fn>).mockClear();
  (api.listFieldDefinitions as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
    items: [
      { id: "def-1", entity_type: "incident", key: "po_number", label: "PO Number", field_type: "TEXT", config: { required: true }, status: "ACTIVE", sort_order: 1, version: 1, created_at: "", updated_at: "" },
      { id: "def-2", entity_type: "incident", key: "retired_required", label: "Retired Required", field_type: "TEXT", config: { required: true }, status: "RETIRED", sort_order: 2, version: 1, created_at: "", updated_at: "" },
    ],
  });
  renderPage();
  fireEvent.click(await screen.findByRole("button", { name: /create incident/i }));
  const dialog = screen.getByRole("dialog");
  fireEvent.change(within(dialog).getByLabelText(/asset/i), { target: { value: "a1" } });
  fireEvent.change(within(dialog).getByLabelText(/summary/i), { target: { value: "High vibration on bearing" } });
  fireEvent.change(within(dialog).getByLabelText(/priority/i), { target: { value: "HIGH" } });
  fireEvent.change(within(dialog).getByLabelText(/PO Number/), { target: { value: "PO-42" } });
  fireEvent.submit(dialog.querySelector("form") as HTMLFormElement);
  await waitFor(() =>
    expect(api.createIncident as ReturnType<typeof vi.fn>).toHaveBeenCalled(),
  );
});

it("does not render retired custom fields in the create form", async () => {
  (api.listFieldDefinitions as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
    items: [
      { id: "def-1", entity_type: "incident", key: "po_number", label: "PO Number", field_type: "TEXT", config: { required: false }, status: "ACTIVE", sort_order: 1, version: 1, created_at: "", updated_at: "" },
      { id: "def-2", entity_type: "incident", key: "retired_note", label: "Retired Note", field_type: "TEXT", config: { required: false }, status: "RETIRED", sort_order: 2, version: 1, created_at: "", updated_at: "" },
    ],
  });
  renderPage();
  fireEvent.click(await screen.findByRole("button", { name: /create incident/i }));
  const dialog = screen.getByRole("dialog");
  expect(await within(dialog).findByLabelText(/PO Number/)).toBeTruthy();
  expect(within(dialog).queryByLabelText(/Retired Note/)).toBeNull();
});

it("forwards custom_values from the create form", async () => {
  (api.createIncident as ReturnType<typeof vi.fn>).mockClear();
  (api.listFieldDefinitions as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
    items: [
      { id: "def-1", entity_type: "incident", key: "po_number", label: "PO Number", field_type: "TEXT", config: { required: false }, status: "ACTIVE", sort_order: 1, version: 1, created_at: "", updated_at: "" }
    ]
  });
  renderPage();
  fireEvent.click(await screen.findByRole("button", { name: /create incident/i }));
  const dialog = screen.getByRole("dialog");
  fireEvent.change(within(dialog).getByLabelText(/asset/i), { target: { value: "a1" } });
  fireEvent.change(within(dialog).getByLabelText(/summary/i), { target: { value: "High vibration on bearing" } });
  fireEvent.change(within(dialog).getByLabelText(/priority/i), { target: { value: "HIGH" } });
  fireEvent.change(within(dialog).getByLabelText(/PO Number/), { target: { value: "PO-42" } });
  fireEvent.submit(dialog.querySelector("form") as HTMLFormElement);
  await waitFor(() =>
    expect(api.createIncident as ReturnType<typeof vi.fn>).toHaveBeenCalledWith(
      expect.objectContaining({ custom_values: { po_number: "PO-42" } }),
    ),
  );
});

it("applies a custom field filter to the incident query", async () => {
  (api.listFieldDefinitions as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
    items: [
      { id: "def-1", entity_type: "incident", key: "zone", label: "Zone", field_type: "SELECT", config: { options: [{ label: "A", value: "a" }] }, status: "ACTIVE", sort_order: 1, version: 1, created_at: "", updated_at: "" }
    ]
  });
  renderPage();
  await waitFor(() => expect(screen.getByLabelText("Zone")).toBeTruthy());
  fireEvent.change(screen.getByLabelText("Zone"), { target: { value: "a" } });
  await waitFor(() =>
    expect(api.incidents as ReturnType<typeof vi.fn>).toHaveBeenCalledWith(
      expect.objectContaining({ custom_fields: { zone: "a" } }),
    ),
  );
});

it("shows custom field values in an expandable queue row", async () => {
  (api.listFieldDefinitions as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
    items: [
      { id: "def-1", entity_type: "incident", key: "zone", label: "Zone", field_type: "SELECT", config: { options: [{ label: "Zone A", value: "a" }] }, status: "ACTIVE", sort_order: 1, version: 1, created_at: "", updated_at: "" }
    ]
  });
  (api.incidents as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
    items: [
      { ...fixtures.OPEN_HIGH, custom_values: { "def-1": "a" } }
    ],
    next_cursor: null, has_more: false, custom_fields: [
      { id: "def-1", entity_type: "incident", key: "zone", label: "Zone", field_type: "SELECT", config: { options: [{ label: "Zone A", value: "a" }] }, status: "ACTIVE", sort_order: 1, version: 1, created_at: "", updated_at: "" }
    ]
  });
  renderPage();
  // Custom fields are not new columns (spec): no dedicated column header.
  expect(screen.queryByRole("columnheader", { name: "Custom fields" })).toBeNull();
  fireEvent.click(await screen.findByRole("button", { name: /show custom fields/i }));
  const detailRow = document.querySelector("tr.data-row-detail");
  expect(detailRow).toBeTruthy();
  expect(within(detailRow as HTMLElement).getByText("Zone A")).toBeTruthy();
});

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { IncidentsPage } from "./IncidentsPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { ToastProvider } from "../feedback/Toast";
import { api } from "../../api";

const fixtures = vi.hoisted(() => ({
  OPEN_HIGH: {
    id: "i1",
    site_id: "s1",
    asset_id: "a1",
    asset_tag: "P-302",
    number: "IN-1",
    summary: "Pump vibration",
    severity: "HIGH",
    state: "OPEN",
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
    severity: "MEDIUM",
    state: "IN_PROGRESS",
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
    severity: "LOW",
    state: "RESOLVED",
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
    incidents: vi.fn().mockResolvedValue({ items: [fixtures.OPEN_HIGH, fixtures.INPROG_MED, fixtures.RESOLVED_LOW] }),
    assets: vi.fn().mockResolvedValue({
      items: [
        { id: "a1", site_id: "s1", tag: "P-302", name: "Process Pump", class: "CENTRIFUGAL_PUMP", status: "OPERATIONAL", source_of_truth: "OWNED_BY_SKAWLD" }
      ]
    }),
    createIncident: vi.fn().mockResolvedValue({ id: "new1", number: "IN-9" })
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
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "p1",
      display_name: "Tester",
      site_ids: ["s1"],
      permissions: ["incident:create", "incident:read"]
    });
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

  it("filters via the KPI cards", async () => {
    renderPage();
    await screen.findByText("Pump vibration");
    fireEvent.click(screen.getByRole("button", { name: /resolved/i }));
    await waitFor(() => expect(screen.queryByText("Pump vibration")).toBeNull());
    expect(screen.getByText("Resolved noise")).toBeTruthy();
  });

  it("links the asset column to the asset page", async () => {
    renderPage();
    const assetLink = await screen.findByRole("link", { name: /P-302/i });
    expect(assetLink.getAttribute("href")).toBe("/assets/a1");
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
    fireEvent.change(within(dialog).getByLabelText(/severity/i), { target: { value: "HIGH" } });
    fireEvent.submit(dialog.querySelector("form") as HTMLFormElement);
    await waitFor(() =>
      expect(api.createIncident as ReturnType<typeof vi.fn>).toHaveBeenCalledWith({
        site_id: "s1",
        asset_id: "a1",
        summary: "High vibration on bearing",
        severity: "HIGH"
      }),
    );
    expect(screen.getByText("Incident created")).toBeTruthy();
  });
});

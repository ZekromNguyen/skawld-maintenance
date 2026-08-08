import { describe, it, expect, vi } from "vitest";
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
    incidents: vi.fn().mockImplementation((options?: { state?: string[]; severity?: string }) => {
      let items = [fixtures.OPEN_HIGH, fixtures.INPROG_MED, fixtures.RESOLVED_LOW];
      if (options?.state?.length) {
        items = items.filter((incident) => options.state!.includes(incident.state));
      }
      if (options?.severity && options.severity !== "ALL") {
        items = items.filter((incident) => incident.severity === options.severity);
      }
      return Promise.resolve({ items, next_cursor: null, has_more: false });
    }),
    summary: vi.fn().mockResolvedValue({
      open_incidents: 1,
      in_progress_incidents: 1,
      resolved_incidents: 1,
      total_incidents: 3,
      by_severity: { LOW: 1, MEDIUM: 1, HIGH: 1, CRITICAL: 0 },
      active_executions: 0,
      critical_assets: 0,
      pending_handovers: 0
    }),
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
    await waitFor(() => expect(screen.queryByText("Pump vibration")).toBeNull());
    expect(screen.getByText("Resolved noise")).toBeTruthy();
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

it("shows load more and appends the next page", async () => {
  const list = vi.mocked(api.incidents);
  list
    .mockResolvedValueOnce({ items: [fixtures.OPEN_HIGH as unknown as Incident], next_cursor: "c1", has_more: true })
    .mockResolvedValueOnce({ items: [fixtures.RESOLVED_LOW as unknown as Incident], next_cursor: null, has_more: false });
  renderPage();
  const button = await screen.findByRole("button", { name: "Load more" });
  fireEvent.click(button);
  await waitFor(() =>
    expect(screen.queryByRole("button", { name: "Load more" })).toBeNull(),
  );
});

describe("IncidentsPage server-side filters", () => {
  it("refetches with the selected severity and filters rows", async () => {
    renderPage();
    await screen.findByText("Pump vibration");
    fireEvent.change(screen.getByLabelText(/severity/i), { target: { value: "HIGH" } });
    await waitFor(() => {
      const calls = (api.incidents as ReturnType<typeof vi.fn>).mock.calls;
      expect(calls.some(([opts]) => opts && opts.severity === "HIGH")).toBe(true);
    });
    await waitFor(() => expect(screen.getByText("Pump vibration")).toBeTruthy());
  });

  it("shows truthful tab counts from the summary endpoint", async () => {
    renderPage();
    await screen.findByText("Pump vibration");
    await waitFor(() => {
      expect(screen.getByRole("tab", { name: /^open/i }).textContent).toContain("1");
    });
  });
});

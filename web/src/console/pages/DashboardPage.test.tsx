import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { DashboardPage } from "./DashboardPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { api } from "../../api";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({
      id: "p1",
      display_name: "Tester",
      site_ids: [],
      permissions: ["incident:read"]
    }),
    incidents: vi.fn().mockResolvedValue({
      items: [
        {
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
        {
          id: "i2",
          site_id: "s1",
          asset_id: "a1",
          asset_tag: "P-302",
          number: "IN-2",
          summary: "Resolved bearing noise",
          severity: "LOW",
          state: "RESOLVED",
          detected_at: new Date().toISOString(),
          version: 1
        }
      ]
    }),
    assets: vi.fn().mockResolvedValue({
      items: [
        {
          id: "a1",
          site_id: "s1",
          tag: "P-302",
          name: "Process Pump",
          class: "CENTRIFUGAL_PUMP",
          status: "OPERATIONAL",
          source_of_truth: "OWNED_BY_SKAWLD",
          criticality: { rating: "A" }
        }
      ]
    }),
    listExecutions: vi.fn().mockResolvedValue({
      items: [
        {
          id: "e1",
          incident_id: "i1",
          asset_id: "a1",
          asset_tag: "P-302",
          purpose: "Shaft alignment check",
          state: "IN_PROGRESS",
          version: 1,
          steps: [],
          measurements: [],
          observations: [],
          actions: []
        }
      ]
    }),
    pendingHandovers: vi.fn().mockResolvedValue([
      {
        id: "h1",
        site_id: "s1",
        shift_start: new Date().toISOString(),
        shift_end: new Date().toISOString(),
        state: "SUBMITTED",
        version: 1,
        structured_content: {},
        evidence: [],
        provider: "skawld-copilot"
      }
    ])
  }
}));

function renderDashboard() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider>
          <MemoryRouter>
            <DashboardPage />
          </MemoryRouter>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("DashboardPage", () => {
  it("renders real KPIs, my queue, and open incidents only", async () => {
    renderDashboard();
    expect(await screen.findByText("Operations overview")).toBeTruthy();
    // Open incident row visible, resolved one excluded.
    expect(screen.getByText("Pump vibration")).toBeTruthy();
    expect(screen.queryByText("Resolved bearing noise")).toBeNull();
    // My queue shows the in-progress execution.
    expect(screen.getByText("Shaft alignment check")).toBeTruthy();
  });

  it("does not render fabricated metrics", async () => {
    renderDashboard();
    await screen.findByText("Operations overview");
    expect(screen.queryByText(/unsafe ai actions/i)).toBeNull();
  });

  it("links KPIs to real console routes", async () => {
    renderDashboard();
    await screen.findByText("Operations overview");
    const incidentsLink = screen.getByRole("link", { name: /open incidents/i });
    expect(incidentsLink.getAttribute("href")).toBe("/incidents");
    const executionsLink = screen.getByRole("link", { name: /executions in progress/i });
    expect(executionsLink.getAttribute("href")).toBe("/executions");
  });

  it("counts pending handovers from the paginated list", async () => {
    renderDashboard();
    const pendingLink = await screen.findByRole("link", { name: /pending handovers/i });
    expect(pendingLink.textContent).toContain("1");
  });
});

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { DashboardPage } from "./DashboardPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
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
        }
      ]
    }),
    assets: vi.fn().mockResolvedValue({ items: [] })
  }
}));

describe("DashboardPage", () => {
  it("renders the dashboard with open incident metrics", async () => {
    render(
      <I18nProvider>
        <PrincipalProvider>
        <MemoryRouter>
          <DashboardPage />
        </MemoryRouter>
        </PrincipalProvider>
      </I18nProvider>,
    );
    expect(await screen.findByText("Operations overview")).toBeTruthy();
    expect(screen.getByText("Pump vibration")).toBeTruthy();
  });

  it("shows supervisor shortcut for open incidents when report:approve present", async () => {
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "p2",
      display_name: "Supervisor",
      site_ids: [],
      permissions: ["report:approve", "incident:read"]
    });
    render(
      <I18nProvider>
        <PrincipalProvider>
        <MemoryRouter>
          <DashboardPage />
        </MemoryRouter>
        </PrincipalProvider>
      </I18nProvider>,
    );
    // "Open incidents" appears in the shortcut card AND the overview metric.
    expect((await screen.findAllByText("Open incidents")).length).toBeGreaterThan(1);
  });
});

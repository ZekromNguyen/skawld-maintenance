import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { MonitoringPage } from "./MonitoringPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { ToastProvider } from "../feedback/Toast";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { api } from "../../api";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({
      id: "p1",
      display_name: "Monitor",
      organization_id: "o1",
      site_ids: ["s1"],
      permissions: ["monitoring:read", "monitoring:configure"],
    }),
    monitoringSummary: vi.fn().mockResolvedValue({
      items: [
        {
          metric_key: "sys.backup_freshness_hours",
          status: "CRIT",
          value: 96,
          unit: "hours",
          group: "system",
          open_alerts: 1,
        },
        {
          metric_key: "sys.health_ready",
          status: "PASS",
          value: 1,
          unit: "0/1",
          group: "system",
          open_alerts: 0,
        },
      ],
    }),
    monitoringAlerts: vi.fn().mockResolvedValue({
      items: [
        {
          id: 1,
          metric_key: "sys.backup_freshness_hours",
          state: "CRIT",
          message: "{}",
          value: 96,
          opened_at: "2026-08-08T00:00:00Z",
        },
      ],
    }),
    monitoringMetrics: vi.fn().mockResolvedValue({
      items: [{ value: 12, sample_count: 1, dimensions: {} }],
    }),
    setMonitoringThreshold: vi.fn().mockResolvedValue(undefined),
  },
}));

function renderPage() {
  return render(
    <I18nProvider>
      <ToastProvider>
        <PrincipalProvider>
          <SiteProvider
          principal={{
            id: "p1",
            display_name: "Monitor",
            organization_id: "o1",
            site_ids: ["s1"],
            permissions: ["monitoring:read", "monitoring:configure"],
          }}
        >
          <MemoryRouter>
            <MonitoringPage />
            </MemoryRouter>
          </SiteProvider>
        </PrincipalProvider>
      </ToastProvider>
    </I18nProvider>,
  );
}

describe("MonitoringPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders status cards with the CRIT badge", async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("sys.backup_freshness_hours")).toBeTruthy();
    });
    expect(screen.getByText("Critical")).toBeTruthy();
    expect(screen.getByText("Pass")).toBeTruthy();
  });

  it("lists alerts in the alerts tab", async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("sys.backup_freshness_hours")).toBeTruthy();
    });
    fireEvent.click(screen.getByText("Alerts"));
    await waitFor(() => {
      expect(screen.getByText("Alerts", { selector: "button" })).toBeTruthy();
    });
  });

  it("saves a threshold through the editor", async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText("sys.backup_freshness_hours")).toBeTruthy();
    });
    fireEvent.click(screen.getByText("Thresholds"));
    await waitFor(() => {
      expect(screen.getByText("Configure")).toBeTruthy();
    });
    fireEvent.click(screen.getByText("Configure"));
    await waitFor(() => {
      expect(screen.getByText("Save")).toBeTruthy();
    });
    fireEvent.click(screen.getByText("Save"));
    await waitFor(() => {
      expect(api.setMonitoringThreshold).toHaveBeenCalled();
    });
  });
});

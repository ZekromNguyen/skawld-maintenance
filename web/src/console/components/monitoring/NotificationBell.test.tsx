import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { NotificationBell } from "./NotificationBell";
import { I18nProvider } from "../../../i18n/I18nProvider";
import { api } from "../../../api";

vi.mock("../../../api", () => ({
  api: {
    monitoringSummary: vi.fn().mockResolvedValue({
      items: [
        {
          metric_key: "sys.backup_freshness_hours",
          status: "CRIT",
          value: 96,
          unit: "hours",
          group: "system",
          open_alerts: 2,
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
  },
}));

describe("NotificationBell", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows the open-alert count and lists CRIT entries", async () => {
    render(
      <I18nProvider>
        <NotificationBell />
      </I18nProvider>,
    );
    await waitFor(() => {
      expect(screen.getByText("2")).toBeTruthy();
    });
    fireEvent.click(screen.getByRole("button", { name: "Alerts" }));
    await waitFor(() => {
      expect(screen.getByText("sys.backup_freshness_hours")).toBeTruthy();
      expect(screen.getByText("Critical")).toBeTruthy();
    });
  });

  it("polls the summary on an interval", async () => {
    vi.useFakeTimers();
    try {
      render(
        <I18nProvider>
          <NotificationBell />
        </I18nProvider>,
      );
      await Promise.resolve();
      await Promise.resolve();
      expect(api.monitoringSummary).toHaveBeenCalledTimes(1);
      await vi.advanceTimersByTimeAsync(60_000);
      expect(api.monitoringSummary).toHaveBeenCalledTimes(2);
    } finally {
      vi.useRealTimers();
    }
  });
});

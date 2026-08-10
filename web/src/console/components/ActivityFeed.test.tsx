import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { ActivityFeed, type ActivityEntry } from "./ActivityFeed";
import { I18nProvider } from "../../i18n/I18nProvider";

function renderFeed(entries: ActivityEntry[]) {
  return render(
    <I18nProvider>
      <MemoryRouter>
        <ActivityFeed entries={entries} />
      </MemoryRouter>
    </I18nProvider>,
  );
}

describe("ActivityFeed", () => {
  it("renders an empty state when there are no entries", () => {
    renderFeed([]);
    expect(screen.getByText("No activity recorded yet.")).toBeTruthy();
  });

  it("renders detected entries with relative time", () => {
    renderFeed([
      {
        id: "a1",
        kind: "detected",
        title: "Incident detected",
        time: new Date().toISOString(),
      },
    ]);
    expect(screen.getByText("Incident detected")).toBeTruthy();
    expect(screen.getByText(/ago$/)).toBeTruthy();
  });

  it("renders execution entries with a link to the workbench", () => {
    renderFeed([
      {
        id: "a2",
        kind: "execution",
        title: "Pump inspection started",
        to: "/executions/e1",
        time: new Date().toISOString(),
      },
    ]);
    const link = screen.getByRole("link", { name: /pump inspection started/i });
    expect(link.getAttribute("href")).toBe("/executions/e1");
  });

  it("renders measurement entries with a mono value", () => {
    renderFeed([
      {
        id: "a3",
        kind: "measurement",
        title: "Vibration velocity",
        detail: "2.4 mm/s",
        time: new Date().toISOString(),
      },
    ]);
    expect(screen.getByText("Vibration velocity")).toBeTruthy();
    expect(screen.getByText("2.4 mm/s")).toBeTruthy();
  });

  it("renders context entries without a timestamp", () => {
    renderFeed([
      {
        id: "a4",
        kind: "context",
        title: "Lockout-tagout step gated",
      },
    ]);
    expect(screen.getByText("Lockout-tagout step gated")).toBeTruthy();
  });
});

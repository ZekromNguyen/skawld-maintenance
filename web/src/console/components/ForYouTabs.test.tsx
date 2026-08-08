import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { ForYouTabs } from "./ForYouTabs";
import { I18nProvider } from "../../i18n/I18nProvider";
import type { Asset, Execution, Incident } from "../../types";

const OPEN_HIGH: Incident = {
  id: "i1",
  site_id: "s1",
  asset_id: "a1",
  asset_tag: "P-302",
  number: "IN-1",
  summary: "Pump vibration",
  severity: "HIGH",
  state: "OPEN",
  detected_at: new Date(Date.now() - 60_000).toISOString(),
  version: 1,
};
const RESOLVED_LOW: Incident = {
  id: "i3",
  site_id: "s1",
  asset_id: "a3",
  asset_tag: "P-305",
  number: "IN-3",
  summary: "Resolved noise",
  severity: "LOW",
  state: "RESOLVED",
  detected_at: new Date().toISOString(),
  version: 1,
};
const INPROG_EXECUTION: Execution = {
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
  actions: [],
};
const ASSIGNED_EXECUTION: Execution = {
  id: "e2",
  incident_id: "",
  asset_id: "a2",
  asset_tag: "P-304",
  purpose: "Bearing inspection",
  state: "ASSIGNED",
  version: 1,
  steps: [],
  measurements: [],
  observations: [],
  actions: [],
};

const ASSET: Asset = {
  id: "a1",
  site_id: "s1",
  tag: "P-302",
  name: "Process Pump",
  class: "CENTRIFUGAL_PUMP",
  status: "OPERATIONAL",
  source_of_truth: "OWNED_BY_SKAWLD",
};

function renderTabs(props: {
  incidents?: Incident[];
  executions?: Execution[];
  assets?: Asset[];
  pendingHandoverCount?: number;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
} = {}) {
  return render(
    <I18nProvider>
      <MemoryRouter>
        <ForYouTabs
          incidents={props.incidents ?? [OPEN_HIGH, RESOLVED_LOW]}
          executions={props.executions ?? [INPROG_EXECUTION, ASSIGNED_EXECUTION]}
          assets={props.assets ?? [ASSET]}
          pendingHandoverCount={props.pendingHandoverCount ?? 0}
          loading={props.loading ?? false}
          error={props.error}
          onRetry={props.onRetry ?? (() => undefined)}
        />
      </MemoryRouter>
    </I18nProvider>,
  );
}

describe("ForYouTabs", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("shows a recommended queue of open incidents with a real count badge", () => {
    renderTabs();
    const recommended = screen.getByRole("tab", { name: /recommended/i });
    expect(recommended.textContent).toContain("1");
    expect(screen.getByText("Pump vibration")).toBeTruthy();
    // resolved incidents are excluded from Recommended
    expect(screen.queryByText("Resolved noise")).toBeNull();
  });

  it("switches to the assigned-to-me queue of in-progress executions", () => {
    renderTabs();
    fireEvent.click(screen.getByRole("tab", { name: /assigned to me/i }));
    expect(screen.getByText("Shaft alignment check")).toBeTruthy();
    // assigned executions count toward the tab badge
    const tab = screen.getByRole("tab", { name: /assigned to me/i });
    expect(tab.textContent).toContain("2");
  });

  it("lists saved views from localStorage under Starred", () => {
    window.localStorage.setItem(
      "skawld.incidents.savedViews",
      JSON.stringify([
        { id: "v1", name: "My critical queue", severity: "CRITICAL", query: "" },
        { id: "v2", name: "Pump focus", severity: "HIGH", query: "pump" },
      ]),
    );
    renderTabs();
    fireEvent.click(screen.getByRole("tab", { name: /starred/i }));
    expect(screen.getByText("My critical queue")).toBeTruthy();
    expect(screen.getByText("Pump focus")).toBeTruthy();
    const tab = screen.getByRole("tab", { name: /starred/i });
    expect(tab.textContent).toContain("2");
  });

  it("shows recently viewed items under Viewed", () => {
    window.localStorage.setItem(
      "skawld.dashboard.recent",
      JSON.stringify([
        { kind: "incident", id: "i1", key: "IN-1", title: "Pump vibration", at: Date.now() },
      ]),
    );
    renderTabs();
    fireEvent.click(screen.getByRole("tab", { name: /viewed/i }));
    expect(screen.getByText("Pump vibration")).toBeTruthy();
  });

  it("records a view when a recommended row is opened", () => {
    renderTabs();
    fireEvent.click(screen.getByText("Pump vibration"));
    const recent = JSON.parse(
      window.localStorage.getItem("skawld.dashboard.recent") ?? "[]",
    ) as Array<{ kind: string; id: string }>;
    expect(recent.some((entry) => entry.id === "i1")).toBe(true);
  });

  it("renders a truthful empty state per tab", () => {
    renderTabs({ incidents: [], executions: [] });
    fireEvent.click(screen.getByRole("tab", { name: /assigned to me/i }));
    expect(screen.getByText("No executions assigned to you right now.")).toBeTruthy();
  });

  it("shows skeletons while loading", () => {
    renderTabs({ loading: true });
    expect(screen.getByRole("status")).toBeTruthy();
  });

  it("shows an error state with retry when data fails", () => {
    const onRetry = vi.fn();
    renderTabs({ error: "boom", onRetry });
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(onRetry).toHaveBeenCalled();
  });

  it("persists the active tab as a saved view preference", () => {
    renderTabs();
    fireEvent.click(screen.getByRole("tab", { name: /assigned to me/i }));
    expect(window.localStorage.getItem("skawld.dashboard.foryouTab")).toBe("assigned");
  });
});

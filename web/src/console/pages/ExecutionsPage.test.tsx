import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { ExecutionsPage } from "./ExecutionsPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { api } from "../../api";
import type { Execution } from "../../types";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({ id: "p1", display_name: "T", site_ids: ["s1"], permissions: [] }),
    listExecutions: vi.fn().mockImplementation((options?: { state?: string[] }) => {
      let items: Execution[] = [
        { id: "e1", incident_id: "i1", incident_number: "IN-1", asset_id: "a1", asset_tag: "P-302", purpose: "Shaft alignment", state: "IN_PROGRESS", version: 1, steps: [{ id: "s1", key: "k", sequence: 1, title: "T", state: "COMPLETED", risk_level: "INFORMATIONAL", version: 1 }], measurements: [], observations: [], actions: [] },
        { id: "e2", incident_id: "i2", incident_number: "IN-2", asset_id: "a2", asset_tag: "P-304", purpose: "Torque check", state: "COMPLETED", version: 1, steps: [], measurements: [], observations: [], actions: [] }
      ];
      if (options?.state?.length) {
        items = items.filter((execution) => options.state!.includes(execution.state));
      }
      return Promise.resolve({ next_cursor: null, has_more: false, items });
    })
  }
}));

function renderPage() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider>
          <MemoryRouter>
            <ExecutionsPage />
          </MemoryRouter>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("ExecutionsPage", () => {
  it("lists executions with links to the workbench", async () => {
    renderPage();
    const shaft = await screen.findByText("Shaft alignment");
    expect(shaft.getAttribute("href")).toBe("/executions/e1");
    expect(screen.getByText("Torque check")).toBeTruthy();
  });

  it("filters by state", async () => {
    renderPage();
    await screen.findByText("Shaft alignment");
    fireEvent.click(screen.getByRole("tab", { name: /completed/i }));
    await waitFor(() => expect(screen.queryByText("Shaft alignment")).toBeNull());
    expect(screen.getByText("Torque check")).toBeTruthy();
    fireEvent.click(screen.getByRole("tab", { name: /all/i }));
    await waitFor(() => expect(screen.getByText("Shaft alignment")).toBeTruthy());
  });
});

it("shows load more and appends the next page", async () => {
  const list = vi.mocked(api.listExecutions);
  const item = { id: "e9", incident_id: "i1", asset_id: "a1", asset_tag: "P-302", purpose: "Shaft alignment", state: "IN_PROGRESS", version: 1, steps: [], measurements: [], observations: [], actions: [] } as unknown as Execution;
  list
    .mockResolvedValueOnce({ items: [item], next_cursor: "c1", has_more: true })
    .mockResolvedValueOnce({ items: [{ ...item, id: "e10" }], next_cursor: null, has_more: false });
  renderPage();
  const button = await screen.findByRole("button", { name: "Load more" });
  fireEvent.click(button);
  await waitFor(() =>
    expect(screen.queryByRole("button", { name: "Load more" })).toBeNull(),
  );
});

describe("ExecutionsPage i18n", () => {
  it("renders execution states via message keys, not raw enums", async () => {
    renderPage();
    await screen.findByText("Shaft alignment");
    expect(screen.getAllByText(/in progress/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/completed/i).length).toBeGreaterThan(0);
    expect(screen.queryByText(/^IN_PROGRESS$/)).toBeNull();
    expect(screen.queryByText(/^COMPLETED$/)).toBeNull();
  });
});

describe("ExecutionsPage incident identity", () => {
  it("shows the incident number in the incident column", async () => {
    renderPage();
    await screen.findByText("Shaft alignment");
    const link = screen.getByRole("link", { name: "IN-1" });
    expect(link.getAttribute("href")).toBe("/incidents/i1");
    expect(screen.getByRole("link", { name: "IN-2" })).toBeTruthy();
  });
});

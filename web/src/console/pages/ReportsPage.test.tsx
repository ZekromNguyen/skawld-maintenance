import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { ReportsPage } from "./ReportsPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { api, type ListOptions } from "../../api";
import type { ListPage, MaintenanceReport } from "../../types";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({ id: "p1", display_name: "T", site_ids: ["s1"], permissions: ["report:write"] }),
    reports: vi.fn().mockResolvedValue({
      items: [
        { id: "r1", execution_id: "e1", revision: 1, version: 1, state: "DRAFT", structured_content: { summary: "Pump inspection", measurements: [], observations: [], actions: [], outcome: "", evidence_ids: [], unknowns: [], requires_human_review: false }, evidence: [] },
        { id: "r2", execution_id: null, revision: 2, version: 1, state: "APPROVED", structured_content: { summary: "Bearing replacement", measurements: [], observations: [], actions: [], outcome: "", evidence_ids: [], unknowns: [], requires_human_review: false }, evidence: [] }
      ],
      next_cursor: null,
      has_more: false
    })
  }
}));

function renderPage() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider>
          <MemoryRouter>
            <ReportsPage />
          </MemoryRouter>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("ReportsPage", () => {
  it("lists reports with tone badges and links", async () => {
    renderPage();
    const reportLink = await screen.findByText("RP-1");
    expect(reportLink.getAttribute("href")).toBe("/reports/r1");
    expect(screen.getByText("Pump inspection")).toBeTruthy();
    expect(screen.getAllByText("Draft").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Approved").length).toBeGreaterThan(0);
  });

  it("links reports to their execution", async () => {
    renderPage();
    const executionLink = await screen.findByRole("link", { name: /^e1$/i });
    expect(executionLink.getAttribute("href")).toBe("/executions/e1");
  });

  it("applies the state filter server-side", async () => {
    renderPage();
    const select = await screen.findByRole("combobox", { name: /state/i });
    fireEvent.change(select, { target: { value: "APPROVED" } });
    await waitFor(() => {
      expect(api.reports).toHaveBeenLastCalledWith({
        state: ["APPROVED"],
        page_size: 25
      });
    });
  });

  it("loads more pages when available", async () => {
    const reportFixture = (id: string, executionId: string, revision: number, state: "DRAFT" | "SUBMITTED" | "APPROVED", summary: string): MaintenanceReport => ({
      id, execution_id: executionId, revision, version: 1, state,
      structured_content: { summary, measurements: [], observations: [], actions: [], outcome: "", evidence_ids: [], unknowns: [], requires_human_review: false },
      evidence: []
    });
    const firstPage: ListPage<MaintenanceReport> = {
      items: [reportFixture("r1", "e1", 1, "DRAFT", "Pump inspection")],
      next_cursor: "c1",
      has_more: true
    };
    const secondPage: ListPage<MaintenanceReport> = {
      items: [reportFixture("r2", "", 2, "APPROVED", "Bearing replacement")],
      next_cursor: null,
      has_more: false
    };
    vi.mocked(api.reports).mockImplementation((options?: ListOptions) =>
      options?.cursor ? Promise.resolve(secondPage) : Promise.resolve(firstPage),
    );
    renderPage();
    const button = await screen.findByRole("button", { name: /load more/i });
    fireEvent.click(button);
    await waitFor(() => {
      expect(screen.getByText("Bearing replacement")).toBeTruthy();
    });
  });
});

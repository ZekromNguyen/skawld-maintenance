import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ReportsPage } from "./ReportsPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { api } from "../../api";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({ id: "p1", display_name: "T", site_ids: ["s1"], permissions: ["report:write"] }),
    reports: vi.fn().mockResolvedValue({
      items: [
        { id: "r1", execution_id: "e1", revision: 1, version: 1, state: "DRAFT", structured_content: { summary: "Pump inspection", measurements: [], observations: [], actions: [], outcome: "", evidence_ids: [], unknowns: [], requires_human_review: false }, evidence: [] },
        { id: "r2", execution_id: null, revision: 2, version: 1, state: "APPROVED", structured_content: { summary: "Bearing replacement", measurements: [], observations: [], actions: [], outcome: "", evidence_ids: [], unknowns: [], requires_human_review: false }, evidence: [] }
      ]
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
    expect(screen.getByText("Draft")).toBeTruthy();
    expect(screen.getByText("Approved")).toBeTruthy();
  });

  it("links reports to their execution", async () => {
    renderPage();
    const executionLink = await screen.findByRole("link", { name: /^e1$/i });
    expect(executionLink.getAttribute("href")).toBe("/executions/e1");
  });
});

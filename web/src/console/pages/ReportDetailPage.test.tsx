import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { ReportDetailPage } from "./ReportDetailPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { api } from "../../api";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({
      id: "p1",
      display_name: "Supervisor",
      site_ids: [],
      permissions: ["report:write", "report:approve"]
    }),
    report: vi.fn().mockResolvedValue({
      id: "rp1",
      execution_id: "ex1",
      revision: 1,
      version: 1,
      state: "DRAFT",
      structured_content: {
        summary: "Inspection found shaft alignment out of spec.",
        measurements: ["8.1 mm/s vibration"],
        observations: ["Bearing housing hot"],
        actions: ["Realigned shaft"],
        outcome: "Awaiting approval",
        evidence_ids: ["e1"],
        unknowns: ["Root cause of drift"],
        requires_human_review: true
      },
      evidence: []
    }),
    submitReport: vi.fn(),
    approveReport: vi.fn()
  }
}));

describe("ReportDetailPage", () => {
  it("renders report content with submit action for DRAFT", async () => {
    render(
      <I18nProvider>
        <PrincipalProvider>
        <MemoryRouter initialEntries={["/reports/rp1"]}>
          <Routes>
            <Route path="/reports/:reportId" element={<ReportDetailPage />} />
          </Routes>
        </MemoryRouter>
        </PrincipalProvider>
      </I18nProvider>,
    );
    expect(await screen.findByText("Inspection found shaft alignment out of spec.")).toBeTruthy();
    expect(screen.getByText("Submit")).toBeTruthy();
  });

  it("renders approve action for SUBMITTED report", async () => {
    (api.report as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "rp2",
      execution_id: "ex1",
      revision: 2,
      version: 1,
      state: "SUBMITTED",
      structured_content: {
        summary: "Second revision summary.",
        measurements: [],
        observations: [],
        actions: [],
        outcome: "Awaiting approval",
        evidence_ids: [],
        unknowns: [],
        requires_human_review: true
      },
      evidence: []
    });
    render(
      <I18nProvider>
        <PrincipalProvider>
        <MemoryRouter initialEntries={["/reports/rp2"]}>
          <Routes>
            <Route path="/reports/:reportId" element={<ReportDetailPage />} />
          </Routes>
        </MemoryRouter>
        </PrincipalProvider>
      </I18nProvider>,
    );
    expect(await screen.findByText("Approve")).toBeTruthy();
  });
});

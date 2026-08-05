import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { ExecutionWorkbenchPage } from "./ExecutionWorkbenchPage";
import { I18nProvider } from "../../i18n/I18nProvider";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({
      id: "p1",
      display_name: "Technician",
      site_ids: ["s1"],
      permissions: ["execution:write", "execution:read"]
    }),
    execution: vi.fn().mockResolvedValue({
      id: "ex1",
      incident_id: "inc1",
      asset_id: "a1",
      asset_tag: "P-302",
      purpose: "High vibration pump inspection",
      state: "IN_PROGRESS",
      version: 2,
      steps: [
        { id: "s1", key: "loto", sequence: 1, title: "Apply LOTO", state: "COMPLETED", risk_level: "SAFETY_SIGNIFICANT", version: 1 },
        { id: "s2", key: "inspect", sequence: 2, title: "Inspect bearing", state: "PENDING", risk_level: "INFORMATIONAL", version: 1 }
      ],
      measurements: [],
      observations: [],
      actions: []
    }),
    startExecution: vi.fn(),
    completeStep: vi.fn(),
    draftReport: vi.fn().mockResolvedValue({ id: "rp1" })
  }
}));

describe("ExecutionWorkbenchPage", () => {
  it("renders steps, LOTO gate, and draft report action", async () => {
    render(
      <I18nProvider>
        <MemoryRouter initialEntries={["/executions/ex1"]}>
          <Routes>
            <Route path="/executions/:executionId" element={<ExecutionWorkbenchPage />} />
          </Routes>
        </MemoryRouter>
      </I18nProvider>,
    );
    expect(await screen.findByText("Apply LOTO")).toBeTruthy();
    expect(screen.getByText("Inspect bearing")).toBeTruthy();
    expect(screen.getByText("Draft report")).toBeTruthy();
  });
});

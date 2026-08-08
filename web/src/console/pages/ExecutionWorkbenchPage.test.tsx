import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { ExecutionWorkbenchPage } from "./ExecutionWorkbenchPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { ToastProvider } from "../feedback/Toast";
import { api } from "../../api";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({
      id: "p1",
      display_name: "Technician",
      site_ids: ["s1"],
      permissions: ["execution:write", "execution:read"]
    }),
    incident: vi.fn().mockResolvedValue({
      id: "inc1",
      site_id: "s1",
      asset_id: "a1",
      asset_tag: "P-302",
      number: "IN-1042",
      summary: "Pump vibration",
      severity: "HIGH",
      state: "IN_PROGRESS",
      detected_at: new Date().toISOString(),
      version: 1
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
        { id: "s1", key: "loto", sequence: 1, title: "Apply LOTO", state: "PENDING", risk_level: "SAFETY_SIGNIFICANT", version: 1 },
        { id: "s2", key: "inspect", sequence: 2, title: "Inspect bearing", state: "PENDING", risk_level: "INFORMATIONAL", version: 1 }
      ],
      measurements: [],
      observations: [{ note: "found wear on inner race" }],
      actions: []
    }),
    startExecution: vi.fn(),
    completeStep: vi.fn().mockResolvedValue({}),
    draftReport: vi.fn().mockResolvedValue({ id: "rp1" })
  }
}));

function renderWorkbench() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider>
          <ToastProvider>
            <MemoryRouter initialEntries={["/executions/ex1"]}>
              <Routes>
                <Route path="/executions/:executionId" element={<ExecutionWorkbenchPage />} />
              </Routes>
            </MemoryRouter>
          </ToastProvider>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("ExecutionWorkbenchPage", () => {
  it("renders steps, observations, and draft report action", async () => {
    renderWorkbench();
    expect(await screen.findByText("Apply LOTO")).toBeTruthy();
    expect(screen.getByText("Inspect bearing")).toBeTruthy();
    expect(screen.getByText("Draft report")).toBeTruthy();
    // Observations are readable, not a raw JSON dump.
    expect(screen.getByText("found wear on inner race")).toBeTruthy();
    expect(document.body.textContent).not.toContain('{"note"');
  });

  it("requires confirmation for safety-significant steps", async () => {
    renderWorkbench();
    const buttons = await screen.findAllByRole("button", { name: "Complete" });
    fireEvent.click(buttons[0]);
    const dialog = screen.getByRole("dialog");
    expect(dialog.textContent).toContain("safety-significant");
    fireEvent.click(within(dialog).getByRole("button", { name: "Complete" }));
    await waitFor(() =>
      expect(api.completeStep as ReturnType<typeof vi.fn>).toHaveBeenCalled(),
    );
    expect(screen.getByText("Step completed")).toBeTruthy();
  });

  it("completes informational steps without confirmation", async () => {
    renderWorkbench();
    const buttons = await screen.findAllByRole("button", { name: "Complete" });
    fireEvent.click(buttons[1]);
    await waitFor(() =>
      expect(api.completeStep as ReturnType<typeof vi.fn>).toHaveBeenCalled(),
    );
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("hides actions for read-only principals", async () => {
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "p2",
      display_name: "Observer",
      site_ids: ["s1"],
      permissions: ["execution:read"]
    });
    renderWorkbench();
    await screen.findByText("Apply LOTO");
    expect(screen.queryByRole("button", { name: "Complete" })).toBeNull();
    expect(screen.queryByText("Draft report")).toBeNull();
  });

  it("shows no LOTO banner when no step is blocked", async () => {
    renderWorkbench();
    await screen.findByText("Apply LOTO");
    expect(screen.queryByText(/LOTO ACTIVE/)).toBeNull();
  });

  it("surfaces a LOTO banner when a step is blocked", async () => {
    (api.execution as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      id: "ex2",
      incident_id: "inc1",
      asset_id: "a1",
      asset_tag: "P-302",
      purpose: "Isolation verification",
      state: "IN_PROGRESS",
      version: 1,
      steps: [
        {
          id: "s9",
          key: "loto",
          sequence: 1,
          title: "Release lockout",
          state: "BLOCKED",
          risk_level: "SAFETY_SIGNIFICANT",
          blocked_reason: "Tagout tag not signed",
          version: 1
        }
      ],
      measurements: [],
      observations: [],
      actions: []
    });
    renderWorkbench();
    expect(await screen.findByText(/LOTO ACTIVE/)).toBeTruthy();
    expect(screen.getByText(/blocking progress/)).toBeTruthy();
  });
});

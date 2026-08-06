import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { ReportDetailPage } from "./ReportDetailPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { ToastProvider } from "../feedback/Toast";
import { api } from "../../api";

const fixtures = vi.hoisted(() => ({
  report: {
    id: "r1",
    execution_id: "e1",
    revision: 1,
    version: 1,
    state: "DRAFT" as const,
    structured_content: {
      summary: "Pump inspection R1",
      measurements: ["8.1 mm/s"],
      observations: [],
      actions: [],
      outcome: "",
      evidence_ids: [],
      unknowns: ["root cause unconfirmed"],
      requires_human_review: true
    },
    evidence: [],
    provider: "skawld-copilot",
    model: "copilot-v1",
    prompt_version: "p17"
  }
}));

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({ id: "p1", display_name: "T", site_ids: ["s1"], permissions: ["report:write", "report:approve"] }),
    report: vi.fn().mockResolvedValue(fixtures.report),
    submitReport: vi.fn().mockResolvedValue({}),
    approveReport: vi.fn().mockResolvedValue({})
  }
}));

function renderDetail() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider>
          <ToastProvider>
            <MemoryRouter initialEntries={["/reports/r1"]}>
              <Routes>
                <Route path="/reports/:reportId" element={<ReportDetailPage />} />
              </Routes>
            </MemoryRouter>
          </ToastProvider>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("ReportDetailPage", () => {
  it("renders content with submit action and human-review banner", async () => {
    renderDetail();
    expect(await screen.findByText("Pump inspection R1")).toBeTruthy();
    expect(screen.getByText("Requires human review")).toBeTruthy();
    expect(screen.getByText("root cause unconfirmed")).toBeTruthy();
    expect(screen.getByText("copilot-v1")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Submit" })).toBeTruthy();
  });

  it("submits with a success toast", async () => {
    renderDetail();
    fireEvent.click(await screen.findByRole("button", { name: "Submit" }));
    await waitFor(() => expect(api.submitReport as ReturnType<typeof vi.fn>).toHaveBeenCalledWith("r1"));
    expect(screen.getByText("Report submitted")).toBeTruthy();
  });

  it("prints via the report action", async () => {
    const printSpy = vi.spyOn(window, "print").mockImplementation(() => {});
    renderDetail();
    fireEvent.click(await screen.findByRole("button", { name: "Print" }));
    expect(printSpy).toHaveBeenCalled();
    printSpy.mockRestore();
  });

  it("confirms before approving a submitted report", async () => {
    (api.report as ReturnType<typeof vi.fn>).mockResolvedValue({ ...fixtures.report, state: "SUBMITTED" });
    renderDetail();
    fireEvent.click(await screen.findByRole("button", { name: "Approve" }));
    const dialog = screen.getByRole("dialog");
    expect(dialog.textContent).toContain("official maintenance record");
    fireEvent.click(within(dialog).getByRole("button", { name: "Approve" }));
    await waitFor(() => expect(api.approveReport as ReturnType<typeof vi.fn>).toHaveBeenCalledWith("r1"));
    expect(screen.getByText("Report approved")).toBeTruthy();
  });

  it("disables submit with a reason without report:write", async () => {
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "p2",
      display_name: "Manager",
      site_ids: ["s1"],
      permissions: ["report:approve", "handover:accept"]
    });
    (api.report as ReturnType<typeof vi.fn>).mockResolvedValue(fixtures.report);
    renderDetail();
    await screen.findByText("Pump inspection R1");
    const submit = screen.getByRole("button", { name: "Submit" }) as HTMLButtonElement;
    expect(submit.disabled).toBe(true);
    expect(submit.title).toBe("Required permission not granted");
  });

  it("disables approve with a reason without report:approve", async () => {
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      id: "p3",
      display_name: "Technician",
      site_ids: ["s1"],
      permissions: ["report:write", "execution:write"]
    });
    (api.report as ReturnType<typeof vi.fn>).mockResolvedValue({ ...fixtures.report, state: "SUBMITTED" });
    renderDetail();
    await screen.findByText("Pump inspection R1");
    const approve = screen.getByRole("button", { name: "Approve" }) as HTMLButtonElement;
    expect(approve.disabled).toBe(true);
    expect(approve.title).toBe("Required permission not granted");
  });
});

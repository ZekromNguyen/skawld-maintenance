import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { DemonstrationsPage } from "./DemonstrationsPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { ToastProvider } from "../feedback/Toast";
import { api } from "../../api";

const fixtures = vi.hoisted(() => ({
  demo: {
    id: "d1",
    site_id: "s1",
    subject_kind: "HANDOVER",
    subject_id: "h1",
    workflow_key: "maintenance.shift_handover",
    schema_version: 1,
    session_id: "ses-1",
    status: "completed" as const,
    review_status: "PENDING" as const,
    initial_context: {},
    events: [
      { id: "e1", ordinal: 1, schema_version: 1, timestamp: new Date().toISOString(), actor_id: "u1", roles: [], source: "SDK", trust: "HIGH", sensitivity: "PUBLIC", action: "inspect_bearing", intent: "Check bearing", entity: { type: "BEARING", id: "b1" }, provenance: [], redactions: [{ json_path: "output.value", reason: "proprietary" }] }
    ],
    capture: { pending: 0, processing: 0, failed: 0, applied: 1 },
    started_at: new Date().toISOString(),
    created_by: "u1"
  }
}));

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: ["demonstration:review", "demonstration:capture"] }),
    demonstrations: vi.fn().mockResolvedValue({ items: [fixtures.demo] }),
    completeDemonstration: vi.fn().mockResolvedValue({}),
    reviewDemonstration: vi.fn().mockResolvedValue({}),
    redactDemonstrationEvent: vi.fn().mockResolvedValue({})
  }
}));

function renderPage() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider principal={{ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: ["demonstration:review", "demonstration:capture"] }}>
          <ToastProvider>
            <MemoryRouter>
              <DemonstrationsPage />
            </MemoryRouter>
          </ToastProvider>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("DemonstrationsPage", () => {
  it("selects a demonstration and shows redaction indicators", async () => {
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: /maintenance.shift_handover/i }));
    expect(screen.getByText("1 redacted")).toBeTruthy();
    expect(screen.getByText("inspect_bearing")).toBeTruthy();
  });

  it("uses a dialog instead of window.prompt for review", async () => {
    const promptSpy = vi.spyOn(window, "prompt").mockImplementation(() => null);
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: /maintenance.shift_handover/i }));
    fireEvent.click(screen.getByRole("button", { name: "Approve trace" }));
    const dialog = screen.getByRole("dialog");
    expect(dialog).toBeTruthy();
    fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: "verified" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Approve trace" }));
    await waitFor(() =>
      expect(api.reviewDemonstration as ReturnType<typeof vi.fn>).toHaveBeenCalledWith("d1", "APPROVED", "verified"),
    );
    expect(promptSpy).not.toHaveBeenCalled();
    promptSpy.mockRestore();
  });

  it("disables review actions with a reason without demonstration:review", async () => {
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      id: "p2",
      display_name: "Technician",
      organization_id: "o1",
      site_ids: ["s1"],
      permissions: ["demonstration:capture", "execution:write"]
    });
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: /maintenance.shift_handover/i }));
    const approve = screen.getByRole("button", { name: "Approve trace" }) as HTMLButtonElement;
    expect(approve.disabled).toBe(true);
    expect(approve.title).toBe("Required permission not granted");
    const reject = screen.getByRole("button", { name: "Reject" }) as HTMLButtonElement;
    expect(reject.disabled).toBe(true);
  });
});

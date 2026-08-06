import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { WorkflowsPage } from "./WorkflowsPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { ToastProvider } from "../feedback/Toast";
import { api } from "../../api";
import type { WorkflowVersion } from "../../types";

const fixtures = vi.hoisted(() => ({
  approvedDemo: {
    id: "d1",
    site_id: "s1",
    subject_kind: "EXECUTION",
    subject_id: "e1",
    workflow_key: "maintenance.high_vibration",
    schema_version: 1,
    session_id: "s1",
    status: "completed" as const,
    review_status: "APPROVED" as const,
    initial_context: {},
    events: [],
    capture: { pending: 0, processing: 0, failed: 0, applied: 1 },
    started_at: new Date().toISOString(),
    created_by: "u1"
  },
  workflow: {
    workflow_id: "w1",
    workflow_key: "maintenance.high_vibration",
    name: "High vibration pump inspection",
    version: 2,
    status: "CANDIDATE" as const,
    site_id: "s1",
    asset_class: "CENTRIFUGAL_PUMP",
    candidate_digest: "abc",
    tool_catalog_digest: "def",
    source_demonstration_ids: ["d1"],
    steps: [],
    analysis: { sequence_consistency: null, conflicts: [], sequence_variants: [], findings: [] },
    behavioral_changes: {},
    learning: {},
    applicability: [],
    prerequisites: [],
    required_competencies: [],
    improvement_candidates: [],
    reviews: [{ decision: "REJECTED" as const, reason: "unsafe step" }],
    evaluations: [],
    created_at: new Date().toISOString()
  }
}));

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: ["workflow:review", "workflow:publish"] }),
    workflows: vi.fn().mockResolvedValue({ items: [fixtures.workflow], next_cursor: null, has_more: false }),
    demonstrations: vi.fn().mockResolvedValue({ items: [fixtures.approvedDemo] }),
    assets: vi.fn().mockResolvedValue({ items: [] }),
    compileWorkflow: vi.fn().mockResolvedValue({}),
    reviewWorkflow: vi.fn().mockResolvedValue({}),
    publishWorkflow: vi.fn().mockResolvedValue({}),
    retireWorkflow: vi.fn().mockResolvedValue({})
  }
}));

function renderPage() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider principal={{ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: ["workflow:review", "workflow:publish"] }}>
          <ToastProvider>
            <MemoryRouter>
              <WorkflowsPage />
            </MemoryRouter>
          </ToastProvider>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("WorkflowsPage", () => {
  it("resolves names from data and shows the governance trail", async () => {
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: /high vibration pump inspection/i }));
    expect(screen.getByText("Governance trail")).toBeTruthy();
    expect(screen.getByText("unsafe step")).toBeTruthy();
    expect(screen.getByText("—")).toBeTruthy();
    expect(screen.getByText("No analysis")).toBeTruthy();
  });

  it("uses a dialog for review instead of window.prompt", async () => {
    const promptSpy = vi.spyOn(window, "prompt").mockImplementation(() => null);
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: /high vibration pump inspection/i }));
    fireEvent.click(screen.getByRole("button", { name: "Approve candidate" }));
    const dialog = screen.getByRole("dialog");
    fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: "looks good" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Approve candidate" }));
    await waitFor(() =>
      expect(api.reviewWorkflow as ReturnType<typeof vi.fn>).toHaveBeenCalled(),
    );
    expect(promptSpy).not.toHaveBeenCalled();
    promptSpy.mockRestore();
  });

  it("disables review actions with a reason without workflow:review", async () => {
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "p2",
      display_name: "Technician",
      organization_id: "o1",
      site_ids: ["s1"],
      permissions: ["execution:write"]
    });
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: /high vibration pump inspection/i }));
    const approve = screen.getByRole("button", { name: "Approve candidate" }) as HTMLButtonElement;
    expect(approve.disabled).toBe(true);
    expect(approve.title).toBe("Required permission not granted");
    const reject = screen.getByRole("button", { name: "Reject" }) as HTMLButtonElement;
    expect(reject.disabled).toBe(true);
  });
});

it("shows load more and appends the next page", async () => {
  const list = vi.mocked(api.workflows);
  list
    .mockResolvedValueOnce({ items: [fixtures.workflow as unknown as WorkflowVersion], next_cursor: "c1", has_more: true })
    .mockResolvedValueOnce({ items: [fixtures.workflow as unknown as WorkflowVersion], next_cursor: null, has_more: false });
  renderPage();
  const button = await screen.findByRole("button", { name: "Load more" });
  fireEvent.click(button);
  await waitFor(() =>
    expect(screen.queryByRole("button", { name: "Load more" })).toBeNull(),
  );
});

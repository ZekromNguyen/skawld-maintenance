import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { HandoverPage } from "./HandoverPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { ToastProvider } from "../feedback/Toast";
import { api } from "../../api";

const fixtures = vi.hoisted(() => ({
  latest: {
    id: "h1",
    site_id: "s1",
    shift_start: "2026-08-06T06:00:00.000Z",
    shift_end: "2026-08-06T18:00:00.000Z",
    state: "DRAFT" as const,
    version: 1,
    structured_content: { summary: "Night shift summary", open_incidents: ["IN-1"], active_executions: [], safety_concerns: [], follow_up: [], evidence_ids: [], unknowns: ["torque spec unverified"], requires_human_review: true },
    evidence: [],
    provider: "skawld-copilot",
    model: "copilot-v1",
    prompt_version: "p17"
  },
  older: {
    id: "h0",
    site_id: "s1",
    shift_start: "2026-08-05T06:00:00.000Z",
    shift_end: "2026-08-05T18:00:00.000Z",
    state: "ACKNOWLEDGED" as const,
    version: 1,
    structured_content: { summary: "Morning shift summary", open_incidents: [], active_executions: [], safety_concerns: [], follow_up: [], evidence_ids: [], unknowns: [], requires_human_review: false },
    evidence: [],
    provider: "skawld-copilot",
    model: "copilot-v1",
    prompt_version: "p17"
  }
}));

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: ["handover:write"] }),
    handovers: vi.fn().mockResolvedValue({ items: [fixtures.latest, fixtures.older], next_cursor: null, has_more: false }),
    submitHandover: vi.fn().mockResolvedValue({}),
    prepareHandover: vi.fn().mockResolvedValue({}),
    startDemonstration: vi.fn().mockResolvedValue({})
  }
}));

function renderPage() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider principal={{ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: ["handover:write"] }}>
          <ToastProvider>
            <MemoryRouter>
              <HandoverPage />
            </MemoryRouter>
          </ToastProvider>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("HandoverPage", () => {
  it("shows the latest handover, safety flag, and history", async () => {
    renderPage();
    expect(await screen.findByText("Night shift summary")).toBeTruthy();
    expect(screen.getByText("Requires human review")).toBeTruthy();
    expect(screen.getByText("torque spec unverified")).toBeTruthy();
    expect(screen.getByText("Previous handovers")).toBeTruthy();
    expect(screen.getByText("Morning shift summary")).toBeTruthy();
  });

  it("submits the draft with a success toast", async () => {
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: "Submit" }));
    await waitFor(() => expect(api.submitHandover as ReturnType<typeof vi.fn>).toHaveBeenCalledWith("h1"));
    expect(screen.getByText("Handover submitted")).toBeTruthy();
  });

  it("prepares a draft scoped to the active site", async () => {
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: /prepare from current records/i }));
    await waitFor(() => expect(api.prepareHandover as ReturnType<typeof vi.fn>).toHaveBeenCalledWith("s1"));
    expect(screen.getByText("Draft prepared")).toBeTruthy();
  });

  it("loads more history when present", async () => {
    const firstPage = { items: [fixtures.latest], next_cursor: "c1", has_more: true };
    const secondPage = { items: [fixtures.older], next_cursor: null, has_more: false };
    vi.mocked(api.handovers).mockImplementation((options?: { cursor?: string }) =>
      options?.cursor ? Promise.resolve(secondPage) : Promise.resolve(firstPage),
    );
    renderPage();
    const button = await screen.findByRole("button", { name: /load more/i });
    fireEvent.click(button);
    await waitFor(() => {
      expect(api.handovers).toHaveBeenLastCalledWith({
        site_id: "s1",
        page_size: 25,
        cursor: "c1",
      });
    });
    expect(await screen.findByText("Morning shift summary")).toBeTruthy();
  });
});

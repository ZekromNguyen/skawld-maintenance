import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { HandoverPage } from "./HandoverPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";

vi.mock("../../api", () => {
  const draftHandover = {
    id: "h1",
    site_id: "s1",
    shift_start: new Date().toISOString(),
    shift_end: new Date().toISOString(),
    state: "DRAFT",
    version: 1,
    structured_content: {
      summary: "Shift overview",
      open_incidents: ["IN-1"],
      active_executions: ["EX-1"],
      safety_concerns: [],
      follow_up: [],
      evidence_ids: [],
      unknowns: [],
      requires_human_review: true
    },
    evidence: [],
    provider: "test",
    model: "test",
    prompt_version: "1"
  };
  return {
    api: {
      principal: vi.fn().mockResolvedValue({
        id: "p1",
        display_name: "Supervisor",
        site_ids: ["s1"],
        permissions: ["handover:write"]
      }),
      handovers: vi.fn().mockResolvedValue({ items: [draftHandover] }),
      submitHandover: vi.fn().mockResolvedValue({ ...draftHandover, state: "SUBMITTED" }),
      acceptHandover: vi.fn(),
      acknowledgeHandover: vi.fn(),
      startDemonstration: vi.fn()
    }
  };
});

describe("HandoverPage", () => {
  it("renders handover with submit transition for DRAFT", async () => {
    render(
      <I18nProvider>
        <PrincipalProvider>
        <MemoryRouter>
          <HandoverPage />
        </MemoryRouter>
        </PrincipalProvider>
      </I18nProvider>,
    );
    expect(await screen.findByText("Shift overview")).toBeTruthy();
    expect(screen.getByText("Submit")).toBeTruthy();
  });
});

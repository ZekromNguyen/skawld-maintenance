import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { I18nProvider } from "../../i18n/I18nProvider";
import { HandoverPanel } from "./HandoverPanel";
import type { ShiftHandover } from "../../types";

function renderPanel(handover?: Partial<ShiftHandover>) {
  const base: ShiftHandover = {
    id: "h1",
    site_id: "s1",
    shift_start: new Date().toISOString(),
    shift_end: new Date().toISOString(),
    state: "DRAFT",
    version: 1,
    structured_content: {
      summary: "Handover summary",
      open_incidents: [{ title: "P-302 vibration", severity: "HIGH" }],
      active_executions: [],
      safety_concerns: [],
      follow_up: [],
      evidence_ids: [],
      unknowns: [],
      requires_human_review: false,
    },
    evidence: [],
    provider: "deterministic",
    model: "mock",
    prompt_version: "v1",
  };
  return render(
    <I18nProvider>
      <HandoverPanel
        handover={{ ...base, ...handover }}
        pending={false}
        canPrepare={false}
        canCapture={false}
        onPrepare={() => {}}
        onCapture={() => {}}
      />
    </I18nProvider>,
  );
}

describe("HandoverPanel", () => {
  it("renders structured open incidents as text, not raw JSON", async () => {
    renderPanel();
    expect(screen.getByText("P-302 vibration")).toBeTruthy();
    expect(screen.queryByText(/\{.*asset_tag/s)).toBeNull();
  });

  it("renders the item detail when present", async () => {
    renderPanel({
      structured_content: {
        summary: "Handover summary",
        open_incidents: [{ title: "P-302 vibration", detail: "High vibes" }],
        active_executions: [],
        safety_concerns: [],
        follow_up: [],
        evidence_ids: [],
        unknowns: [],
        requires_human_review: false,
      },
    });
    expect(screen.getByText("P-302 vibration")).toBeTruthy();
    expect(screen.getByText(/High vibes/)).toBeTruthy();
  });
});

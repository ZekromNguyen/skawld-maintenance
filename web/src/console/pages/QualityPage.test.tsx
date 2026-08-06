import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QualityPage } from "./QualityPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { api } from "../../api";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: [] }),
    evaluationSummary: vi.fn().mockResolvedValue({
      recommendations: 100,
      reviewed: 90,
      review_coverage: 0.9,
      evidence_coverage: 0.85,
      unsafe_recommendation_rate: 0,
      workflow_gate_pass_rate: 1,
      workflow_evaluations: 5,
      recommendation_acceptance: 0.7,
      human_override_rate: 0.05,
      unsupported_recommendation_rate: 0.02,
      incorrect_next_step_rate: 0.01,
      retrieval_precision: 0.95,
      llm_calls: 1234,
      average_latency_ms: 850,
      tokens_in: 100000,
      tokens_out: 20000,
      estimated_cost_micros: 1234567,
      generated_at: new Date().toISOString()
    })
  }
}));

function renderPage() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider principal={{ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: [] }}>
          <MemoryRouter>
            <QualityPage />
          </MemoryRouter>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("QualityPage", () => {
  it("formats percentages, currency, and tokens correctly", async () => {
    renderPage();
    expect(await screen.findByText("90%")).toBeTruthy();
    expect(screen.getByText("0%")).toBeTruthy();
    expect(screen.getByText("$1.23")).toBeTruthy();
    expect(screen.getByText("120,000")).toBeTruthy();
    expect(screen.queryByText("00%")).toBeNull();
  });

  it("shows last updated and refreshes", async () => {
    renderPage();
    const refresh = await screen.findByRole("button", { name: "Refresh" });
    fireEvent.click(refresh);
    await waitFor(() => expect(api.evaluationSummary as ReturnType<typeof vi.fn>).toHaveBeenCalled());
  });
});

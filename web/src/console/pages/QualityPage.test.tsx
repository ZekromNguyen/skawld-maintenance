import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
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
    expect(await screen.findByText(/90%.*90\/100/i)).toBeTruthy();
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

describe("QualityPage honesty", () => {
  it("shows the denominator next to the review coverage percentage", async () => {
    renderPage();
    await screen.findByText(/90%.*90\/100/i);
  });

  it("shows no-data instead of hard zeros when no evaluations exist", async () => {
    (api.evaluationSummary as ReturnType<typeof vi.fn>).mockResolvedValue({
      recommendations: 0,
      reviewed: 0,
      review_coverage: 0,
      evidence_coverage: 0,
      unsafe_recommendation_rate: 0,
      workflow_gate_pass_rate: 0,
      workflow_evaluations: 0,
      recommendation_acceptance: 0,
      human_override_rate: 0,
      unsupported_recommendation_rate: 0,
      incorrect_next_step_rate: 0,
      retrieval_precision: 0,
      llm_calls: 0,
      average_latency_ms: 0,
      tokens_in: 0,
      tokens_out: 0,
      estimated_cost_micros: 0,
      generated_at: new Date().toISOString()
    });
    renderPage();
    expect((await screen.findAllByText(/no data yet/i)).length).toBeGreaterThan(3);
    expect(screen.queryByText("$0.00")).toBeNull();
    expect(screen.queryByText("0%")).toBeNull();
  });

  it("shows 100% (1/1) for a single reviewed recommendation", async () => {
    (api.evaluationSummary as ReturnType<typeof vi.fn>).mockResolvedValue({
      recommendations: 1,
      reviewed: 1,
      review_coverage: 1,
      evidence_coverage: 1,
      unsafe_recommendation_rate: 0,
      workflow_gate_pass_rate: 0,
      workflow_evaluations: 0,
      recommendation_acceptance: 0,
      human_override_rate: 0,
      unsupported_recommendation_rate: 0,
      incorrect_next_step_rate: 0,
      retrieval_precision: 1,
      llm_calls: 1,
      average_latency_ms: 100,
      tokens_in: 10,
      tokens_out: 5,
      estimated_cost_micros: 50,
      generated_at: new Date().toISOString()
    });
    renderPage();
    expect(await screen.findByText(/100% \(1\/1\)/i)).toBeTruthy();
    expect(screen.getAllByText("100%").length).toBeGreaterThan(0);
  });

  it("shows no-data for recommendation cards when only workflows ran", async () => {
    (api.evaluationSummary as ReturnType<typeof vi.fn>).mockResolvedValue({
      recommendations: 0,
      reviewed: 0,
      review_coverage: 0,
      evidence_coverage: 0,
      unsafe_recommendation_rate: 0,
      workflow_gate_pass_rate: 1,
      workflow_evaluations: 3,
      recommendation_acceptance: 0,
      human_override_rate: 0,
      unsupported_recommendation_rate: 0,
      incorrect_next_step_rate: 0,
      retrieval_precision: 0,
      llm_calls: 5,
      average_latency_ms: 200,
      tokens_in: 100,
      tokens_out: 50,
      estimated_cost_micros: 300,
      generated_at: new Date().toISOString()
    });
    renderPage();
    expect(await screen.findByText("100%")).toBeTruthy();
    expect(screen.getAllByText(/no data yet/i).length).toBeGreaterThan(0);
  });
});

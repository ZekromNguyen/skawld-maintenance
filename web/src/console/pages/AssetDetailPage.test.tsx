import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { AssetDetailPage } from "./AssetDetailPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { ToastProvider } from "../feedback/Toast";
import { api } from "../../api";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({
      id: "p1",
      display_name: "T",
      site_ids: ["s1"],
      permissions: ["asset:criticality:approve"]
    }),
    asset: vi.fn().mockResolvedValue({
      id: "a1",
      site_id: "s1",
      tag: "P-302",
      name: "Process Pump",
      class: "CENTRIFUGAL_PUMP",
      manufacturer: "Grundfos",
      status: "OPERATIONAL",
      source_of_truth: "OWNED_BY_SKAWLD",
      criticality: { rating: "A", safety_impact: "HIGH", production_impact: "MEDIUM", rationale: "Single point of failure" }
    }),
    incidents: vi.fn().mockResolvedValue({
      items: [
        { id: "i1", site_id: "s1", asset_id: "a1", asset_tag: "P-302", number: "IN-1", summary: "Pump vibration", severity: "HIGH", state: "OPEN", detected_at: new Date().toISOString(), version: 1 }
      ]
    }),
    listExecutions: vi.fn().mockResolvedValue({ items: [] }),
    applicableWorkflows: vi.fn().mockResolvedValue({ items: [] }),
    approveAssetCriticality: vi.fn().mockResolvedValue({})
  }
}));

function renderDetail() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider>
          <ToastProvider>
            <MemoryRouter initialEntries={["/assets/a1"]}>
              <Routes>
                <Route path="/assets/:assetId" element={<AssetDetailPage />} />
              </Routes>
            </MemoryRouter>
          </ToastProvider>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("AssetDetailPage", () => {
  it("renders identity, criticality, and linked history", async () => {
    renderDetail();
    expect(await screen.findByText("Process Pump")).toBeTruthy();
    expect(screen.getByText("Skawld native")).toBeTruthy();
    expect(screen.getByText("Pump vibration")).toBeTruthy();
    expect(screen.getByText("Single point of failure")).toBeTruthy();
  });

  it("approves criticality with feedback", async () => {
    renderDetail();
    fireEvent.click(await screen.findByRole("button", { name: /approve criticality/i }));
    await waitFor(() =>
      expect(api.approveAssetCriticality as ReturnType<typeof vi.fn>).toHaveBeenCalledWith("a1"),
    );
    expect(screen.getByText("Criticality approved")).toBeTruthy();
  });

  it("hides approve without permission", async () => {
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "p2",
      display_name: "T",
      site_ids: ["s1"],
      permissions: []
    });
    renderDetail();
    await screen.findByText("Process Pump");
    expect(screen.queryByRole("button", { name: /approve criticality/i })).toBeNull();
  });
});

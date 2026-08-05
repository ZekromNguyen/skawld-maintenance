import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { AssetDetailPage } from "./AssetDetailPage";
import { I18nProvider } from "../../i18n/I18nProvider";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({
      id: "p1",
      display_name: "Tester",
      site_ids: [],
      permissions: []
    }),
    asset: vi.fn().mockResolvedValue({
      id: "a1",
      site_id: "s1",
      tag: "P-302",
      name: "Circulation Pump",
      class: "CENTRIFUGAL_PUMP",
      status: "OPERATIONAL",
      source_of_truth: "OWNED_BY_SKAWLD",
      criticality: {
        rating: "A",
        safety_impact: 5,
        production_impact: 4,
        rationale: "Critical to process"
      }
    })
  }
}));

describe("AssetDetailPage", () => {
  it("renders asset tag and criticality", async () => {
    render(
      <I18nProvider>
        <MemoryRouter initialEntries={["/assets/a1"]}>
          <Routes>
            <Route path="/assets/:assetId" element={<AssetDetailPage />} />
          </Routes>
        </MemoryRouter>
      </I18nProvider>,
    );
    // P-302 appears in the breadcrumb and the page title.
    expect((await screen.findAllByText("P-302")).length).toBeGreaterThan(0);
    expect(screen.getByText("Circulation Pump")).toBeTruthy();
    expect(screen.getByText("A")).toBeTruthy();
  });
});

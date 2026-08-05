import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { SearchPage } from "./SearchPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { api } from "../../api";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({
      id: "p1",
      display_name: "Technician",
      site_ids: ["s1"],
      permissions: []
    }),
    searchKnowledge: vi.fn().mockResolvedValue({
      retrieval_run_id: "run1",
      items: [
        { id: "e1", kind: "document", source_id: "d1", title: "SOP pump alignment", locator: "R2", authority: "SITE_APPROVED", content: "Align shaft within 0.05 mm", content_sha256: "x", score: { rrf_score: 0.9 } },
        { id: "e2", kind: "document", source_id: "d2", title: "Incident IN-1042", locator: "report", authority: "RESOLVED_INCIDENT", content: "Vibration 8.1 mm/s", content_sha256: "y", score: { rrf_score: 0.7 } }
      ]
    })
  }
}));

describe("SearchPage", () => {
  it("renders query and groups evidence by authority", async () => {
    render(
      <I18nProvider>
        <MemoryRouter initialEntries={["/search?q=pump"]}>
          <Routes>
            <Route path="/search" element={<SearchPage />} />
          </Routes>
        </MemoryRouter>
      </I18nProvider>,
    );
    expect(await screen.findByText("SOP pump alignment")).toBeTruthy();
    expect(screen.getByText("Incident IN-1042")).toBeTruthy();
    expect(screen.getByText("SITE_APPROVED")).toBeTruthy();
    expect(screen.getByText("RESOLVED_INCIDENT")).toBeTruthy();
  });
});

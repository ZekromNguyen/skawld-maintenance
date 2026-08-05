import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { DocumentDetailPage } from "./DocumentDetailPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { api } from "../../api";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({
      id: "p1",
      display_name: "Supervisor",
      site_ids: [],
      permissions: ["knowledge:approve", "knowledge:write"]
    }),
    document: vi.fn().mockResolvedValue({
      id: "doc1",
      site_id: "s1",
      document_type: "SOP",
      title: "Pump alignment procedure",
      authority: "SITE_APPROVED",
      revisions: [
        {
          id: "rev1",
          document_id: "doc1",
          revision: "R1",
          approval_status: "APPROVED",
          ingestion_state: "READY",
          language: "en",
          version: 1,
          applicability: [{ site_id: "s1" }]
        },
        {
          id: "rev2",
          document_id: "doc1",
          revision: "R2",
          approval_status: "DRAFT",
          ingestion_state: "AWAITING_UPLOAD",
          language: "en",
          version: 2,
          applicability: []
        }
      ]
    })
  }
}));

describe("DocumentDetailPage", () => {
  it("renders document title and revisions with approve action", async () => {
    render(
      <I18nProvider>
        <PrincipalProvider>
        <MemoryRouter initialEntries={["/knowledge/doc1"]}>
          <Routes>
            <Route path="/knowledge/:documentId" element={<DocumentDetailPage />} />
          </Routes>
        </MemoryRouter>
        </PrincipalProvider>
      </I18nProvider>,
    );
    // Title appears in breadcrumb and topbar.
    expect((await screen.findAllByText("Pump alignment procedure")).length).toBeGreaterThan(0);
    expect(screen.getByText("R1")).toBeTruthy();
    expect(screen.getByText("R2")).toBeTruthy();
    expect(screen.getByText("Approve")).toBeTruthy();
  });
});

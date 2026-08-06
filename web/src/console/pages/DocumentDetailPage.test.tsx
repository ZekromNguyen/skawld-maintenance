import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { DocumentDetailPage } from "./DocumentDetailPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { ToastProvider } from "../feedback/Toast";
import { api } from "../../api";

const fixtures = vi.hoisted(() => ({
  document: {
    id: "d1",
    site_id: "s1",
    document_type: "SOP",
    title: "LOTO Procedure",
    authority: "SITE_APPROVED",
    source_reference: "ref-1",
    revisions: [
      { id: "r1", document_id: "d1", revision: "R1", approval_status: "DRAFT" as const, ingestion_state: "AWAITING_UPLOAD" as const, language: "en", version: 3, applicability: [{ site_id: "s1", asset_class: "CENTRIFUGAL_PUMP" }] },
      { id: "r2", document_id: "d1", revision: "R2", approval_status: "APPROVED" as const, ingestion_state: "READY" as const, language: "en", version: 1, applicability: [] }
    ]
  }
}));

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({ id: "p1", display_name: "T", site_ids: ["s1"], permissions: ["knowledge:approve", "knowledge:write"] }),
    document: vi.fn().mockResolvedValue(fixtures.document),
    approveDocumentRevision: vi.fn().mockResolvedValue({}),
    retireDocumentRevision: vi.fn().mockResolvedValue({}),
    requestDocumentIngestion: vi.fn().mockResolvedValue({})
  }
}));

function renderDetail() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider>
          <ToastProvider>
            <MemoryRouter initialEntries={["/knowledge/d1"]}>
              <Routes>
                <Route path="/knowledge/:documentId" element={<DocumentDetailPage />} />
              </Routes>
            </MemoryRouter>
          </ToastProvider>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("DocumentDetailPage", () => {
  it("renders revisions with tone badges and human-readable applicability", async () => {
    renderDetail();
    expect(await screen.findByRole("heading", { level: 2, name: "LOTO Procedure" })).toBeTruthy();
    expect(screen.getByText("Draft")).toBeTruthy();
    expect(screen.getByText("Approved")).toBeTruthy();
    expect(screen.getByText(/s1 · CENTRIFUGAL_PUMP/)).toBeTruthy();
  });

  it("confirms before approving a draft revision", async () => {
    renderDetail();
    fireEvent.click(await screen.findByRole("button", { name: "Approve" }));
    const dialog = screen.getByRole("dialog");
    expect(dialog.textContent).toContain("effective procedure");
    fireEvent.click(within(dialog).getByRole("button", { name: "Approve" }));
    await waitFor(() => expect(api.approveDocumentRevision as ReturnType<typeof vi.fn>).toHaveBeenCalledWith("r1"));
    expect(screen.getByText("Revision approved")).toBeTruthy();
  });

  it("offers retire only on the approved revision", async () => {
    renderDetail();
    await screen.findByRole("heading", { level: 2, name: "LOTO Procedure" });
    expect(screen.getAllByRole("button", { name: "Retire" }).length).toBe(1);
  });
});

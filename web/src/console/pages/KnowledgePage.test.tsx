import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { KnowledgePage } from "./KnowledgePage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { ToastProvider } from "../feedback/Toast";
import { api } from "../../api";
import type { KnowledgeDocument } from "../../types";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({
      id: "p1",
      display_name: "T",
      site_ids: ["s1"],
      permissions: ["knowledge:write"]
    }),
    documents: vi.fn().mockResolvedValue({ items: [], next_cursor: null, has_more: false }),
    createDocument: vi.fn().mockResolvedValue({}),
    createDocumentRevision: vi.fn().mockResolvedValue({}),
    uploadDocumentRevision: vi.fn().mockResolvedValue({})
  }
}));

function renderPage() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider>
          <ToastProvider>
            <MemoryRouter>
              <KnowledgePage />
            </MemoryRouter>
          </ToastProvider>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("KnowledgePage", () => {
  it("shows load more and appends the next page", async () => {
    const list = vi.mocked(api.documents);
    const document = {
      id: "d1",
      site_id: "s1",
      document_type: "SOP",
      title: "Pump SOP",
      authority: "SITE_APPROVED",
      revisions: []
    };
    list
      .mockResolvedValueOnce({ items: [document], next_cursor: "c1", has_more: true })
      .mockResolvedValueOnce({ items: [document], next_cursor: "c2", has_more: true })
      .mockResolvedValueOnce({ items: [{ ...document, id: "d2", title: "Pump SOP 2" }], next_cursor: null, has_more: false });
    renderPage();
    const button = await screen.findByRole("button", { name: "Load more" });
    fireEvent.click(button);
    await waitFor(() =>
      expect(screen.queryByRole("button", { name: "Load more" })).toBeNull(),
    );
  });
});

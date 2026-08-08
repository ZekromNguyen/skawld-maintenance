import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { SearchPage } from "./SearchPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { api } from "../../api";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({ id: "p1", display_name: "T", site_ids: ["s1"], permissions: [] }),
    searchKnowledge: vi.fn().mockResolvedValue({
      retrieval_run_id: "rr1",
      items: [
        { id: "e1", kind: "DOCUMENT", source_id: "d9", title: "LOTO Procedure for P-302", locator: "sop/loto", authority: "SITE_APPROVED", content: "Apply lockout before inspection.", content_sha256: "x", score: { rrf_score: 0.94 } },
        { id: "e2", kind: "INCIDENT", source_id: "i7", title: "Pump vibration incident", locator: "inc/7", authority: "SITE_APPROVED", content: "Vibration readings above threshold.", content_sha256: "y", score: { rrf_score: 0.5 } }
      ]
    })
  }
}));

function renderPage(initialEntry = "/search") {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider principal={{ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: [] }}>
          <MemoryRouter initialEntries={[initialEntry]}>
            <SearchPage />
          </MemoryRouter>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("SearchPage", () => {
  beforeEach(() => {
    (api.searchKnowledge as ReturnType<typeof vi.fn>).mockClear();
  });

  it("shows an empty state without a query", () => {
    renderPage();
    expect(screen.getByRole("heading", { level: 1, name: "Search" })).toBeTruthy();
  });

  it("renders results that navigate to their owning surface", async () => {
    renderPage("/search?q=loto");
    const docLink = await screen.findByRole("link", { name: /loto procedure/i });
    expect(docLink.getAttribute("href")).toBe("/knowledge/d9");
    const incidentLink = screen.getByRole("link", { name: /pump vibration incident/i });
    expect(incidentLink.getAttribute("href")).toBe("/incidents/i7");
    expect(api.searchKnowledge as ReturnType<typeof vi.fn>).toHaveBeenCalledWith("s1", "loto", undefined, 20);
  });

  it("submits the query into the URL", async () => {
    renderPage();
    const input = screen.getByRole("searchbox");
    fireEvent.change(input, { target: { value: "bearing" } });
    const form = screen.getByRole("button", { name: /search/i }).closest("form");
    fireEvent.submit(form as HTMLFormElement);
    await waitFor(() =>
      expect(api.searchKnowledge as ReturnType<typeof vi.fn>).toHaveBeenCalledWith("s1", "bearing", undefined, 20),
    );
  });
});

describe("SearchPage affordances", () => {
  beforeEach(() => {
    localStorage.clear();
    (api.searchKnowledge as ReturnType<typeof vi.fn>).mockClear();
  });

  it("autofocuses the search input", () => {
    renderPage();
    expect(document.activeElement).toBe(screen.getByRole("searchbox"));
  });

  it("stores recent queries and renders them as shortcuts", async () => {
    renderPage();
    const input = screen.getByRole("searchbox");
    fireEvent.change(input, { target: { value: "bearing" } });
    fireEvent.submit(input.closest("form") as HTMLFormElement);
    await waitFor(() =>
      expect(JSON.parse(localStorage.getItem("skawld.search.recent") ?? "[]")).toContain("bearing"),
    );
    const { unmount } = renderPage();
    unmount();
    renderPage();
    expect(screen.getByRole("link", { name: /bearing/i })).toBeTruthy();
  });

  it("echoes the query when there are no results", async () => {
    (api.searchKnowledge as ReturnType<typeof vi.fn>).mockResolvedValue({
      retrieval_run_id: "rr0", items: []
    });
    renderPage("/search?q=zzzz");
    expect(await screen.findByText(/zzzz/i)).toBeTruthy();
  });
});

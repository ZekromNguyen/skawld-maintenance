import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ExecutionsPage } from "./ExecutionsPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { api } from "../../api";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({ id: "p1", display_name: "T", site_ids: ["s1"], permissions: [] }),
    listExecutions: vi.fn().mockResolvedValue({
      items: [
        { id: "e1", incident_id: "i1", asset_id: "a1", asset_tag: "P-302", purpose: "Shaft alignment", state: "IN_PROGRESS", version: 1, steps: [{ id: "s1", key: "k", sequence: 1, title: "T", state: "COMPLETED", risk_level: "INFORMATIONAL", version: 1 }], measurements: [], observations: [], actions: [] },
        { id: "e2", incident_id: "i2", asset_id: "a2", asset_tag: "P-304", purpose: "Torque check", state: "COMPLETED", version: 1, steps: [], measurements: [], observations: [], actions: [] }
      ]
    })
  }
}));

function renderPage() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider>
          <MemoryRouter>
            <ExecutionsPage />
          </MemoryRouter>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("ExecutionsPage", () => {
  it("lists executions with links to the workbench", async () => {
    renderPage();
    const shaft = await screen.findByText("Shaft alignment");
    expect(shaft.getAttribute("href")).toBe("/executions/e1");
    expect(screen.getByText("Torque check")).toBeTruthy();
  });

  it("filters by state", async () => {
    renderPage();
    await screen.findByText("Shaft alignment");
    fireEvent.click(screen.getByRole("tab", { name: /completed/i }));
    await waitFor(() => expect(screen.queryByText("Shaft alignment")).toBeNull());
    expect(screen.getByText("Torque check")).toBeTruthy();
    fireEvent.click(screen.getByRole("tab", { name: /all/i }));
    expect(screen.getByText("Shaft alignment")).toBeTruthy();
  });
});

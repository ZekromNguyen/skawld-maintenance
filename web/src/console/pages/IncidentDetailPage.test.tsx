import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { IncidentDetailPage } from "./IncidentDetailPage";
import { I18nProvider } from "../../i18n/I18nProvider";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({
      id: "p1",
      display_name: "Supervisor",
      site_ids: ["s1"],
      permissions: ["execution:write", "incident:read", "incident:resolve"]
    }),
    incident: vi.fn().mockResolvedValue({
      id: "inc1",
      site_id: "s1",
      asset_id: "a1",
      asset_tag: "P-302",
      number: "IN-1042",
      summary: "High vibration on pump",
      severity: "HIGH",
      state: "OPEN",
      detected_at: new Date().toISOString(),
      version: 1
    }),
    listExecutions: vi.fn().mockResolvedValue({ items: [] })
  }
}));

describe("IncidentDetailPage", () => {
  it("renders incident summary, severity, and create-execution action", async () => {
    render(
      <I18nProvider>
        <MemoryRouter initialEntries={["/incidents/inc1"]}>
          <Routes>
            <Route path="/incidents/:incidentId" element={<IncidentDetailPage />} />
          </Routes>
        </MemoryRouter>
      </I18nProvider>,
    );
    expect(await screen.findByText("High vibration on pump")).toBeTruthy();
    expect(screen.getByText("HIGH")).toBeTruthy();
    expect(screen.getByText("Create execution")).toBeTruthy();
    expect(screen.getByText("Resolve incident")).toBeTruthy();
  });
});

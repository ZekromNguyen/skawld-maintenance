import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { IncidentDetailPage } from "./IncidentDetailPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { SiteProvider } from "../state/SiteContext";
import { ToastProvider } from "../feedback/Toast";
import { api } from "../../api";

vi.mock("../../api", () => ({
  api: {
    principal: vi.fn().mockResolvedValue({
      id: "p1",
      display_name: "Supervisor",
      site_ids: ["s1"],
      permissions: ["execution:write", "incident:read", "incident:resolve", "recommendation:run"]
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
    listExecutions: vi.fn().mockResolvedValue({ items: [] }),
    resolveIncident: vi.fn().mockResolvedValue({}),
    generateRecommendation: vi.fn().mockResolvedValue({})
  }
}));

function renderDetail() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider>
          <ToastProvider>
            <MemoryRouter initialEntries={["/incidents/inc1"]}>
              <Routes>
                <Route path="/incidents/:incidentId" element={<IncidentDetailPage />} />
              </Routes>
            </MemoryRouter>
          </ToastProvider>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("IncidentDetailPage", () => {
  it("renders incident facts and actions", async () => {
    renderDetail();
    expect(await screen.findByText("High vibration on pump")).toBeTruthy();
    expect(screen.getByText("High")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Create execution" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Resolve incident" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Generate evidence-backed recommendation" })).toBeTruthy();
  });

  it("requires confirmation before resolving", async () => {
    renderDetail();
    fireEvent.click(await screen.findByRole("button", { name: "Resolve incident" }));
    const dialog = screen.getByRole("dialog");
    expect(dialog.textContent).toContain("cannot be undone");
    fireEvent.click(within(dialog).getByRole("button", { name: "Resolve incident" }));
    await waitFor(() =>
      expect(api.resolveIncident as ReturnType<typeof vi.fn>).toHaveBeenCalledWith("inc1"),
    );
    expect(screen.getByText("Incident resolved")).toBeTruthy();
  });

  it("disables resolve with a reason for unauthorized principals", async () => {
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "p2",
      display_name: "Tech",
      site_ids: ["s1"],
      permissions: ["execution:write"]
    });
    renderDetail();
    await screen.findByText("High vibration on pump");
    const resolve = screen.getByRole("button", { name: "Resolve incident" }) as HTMLButtonElement;
    expect(resolve.disabled).toBe(true);
    expect(resolve.title).toBe("Required permission not granted");
  });

  it("generates a recommendation when permitted", async () => {
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      id: "p1",
      display_name: "Supervisor",
      site_ids: ["s1"],
      permissions: ["execution:write", "incident:read", "incident:resolve", "recommendation:run"]
    });
    renderDetail();
    fireEvent.click(await screen.findByRole("button", { name: "Generate evidence-backed recommendation" }));
    await waitFor(() =>
      expect(api.generateRecommendation as ReturnType<typeof vi.fn>).toHaveBeenCalledWith("inc1"),
    );
    expect(screen.getByText("Recommendation generated")).toBeTruthy();
  });

  it("disables recommendation generation without recommendation:run", async () => {
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "p3",
      display_name: "Tech",
      site_ids: ["s1"],
      permissions: ["execution:write", "incident:resolve"]
    });
    renderDetail();
    await screen.findByText("High vibration on pump");
    const button = screen.getByRole("button", { name: "Generate evidence-backed recommendation" }) as HTMLButtonElement;
    expect(button.disabled).toBe(true);
    expect(button.title).toBe("Required permission not granted");
  });
});

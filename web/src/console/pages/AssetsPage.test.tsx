import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { AssetsPage } from "./AssetsPage";
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
      site_ids: ["s1", "s2"],
      permissions: ["asset:create"]
    }),
    assets: vi.fn().mockResolvedValue({
      items: [
        { id: "a1", site_id: "s1", tag: "P-302", name: "Process Pump", class: "CENTRIFUGAL_PUMP", status: "OPERATIONAL", source_of_truth: "OWNED_BY_SKAWLD" },
        { id: "a2", site_id: "s2", tag: "P-304", name: "Bearing Housing", class: "BEARING", status: "DECOMMISSIONED", source_of_truth: "EXTERNAL_REFERENCE" }
      ]
    }),
    createAsset: vi.fn().mockResolvedValue({})
  }
}));

function renderPage() {
  return render(
    <I18nProvider>
      <PrincipalProvider>
        <SiteProvider>
          <ToastProvider>
            <MemoryRouter>
              <AssetsPage />
            </MemoryRouter>
          </ToastProvider>
        </SiteProvider>
      </PrincipalProvider>
    </I18nProvider>,
  );
}

describe("AssetsPage", () => {
  beforeEach(() => {
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "p1",
      display_name: "T",
      site_ids: ["s1", "s2"],
      permissions: ["asset:create"]
    });
  });

  it("renders the registry with localized source badges", async () => {
    renderPage();
    expect(await screen.findByText("Process Pump")).toBeTruthy();
    expect(screen.getByText("P-304")).toBeTruthy();
    expect(screen.getByText("Skawld native")).toBeTruthy();
    expect(screen.getByText("External projection")).toBeTruthy();
  });

  it("hides Add asset for unauthorized principals", async () => {
    (api.principal as ReturnType<typeof vi.fn>).mockResolvedValue({
      id: "p2",
      display_name: "Reader",
      site_ids: ["s1"],
      permissions: []
    });
    renderPage();
    await screen.findByText("Process Pump");
    expect(screen.queryByRole("button", { name: /add asset/i })).toBeNull();
  });

  it("validates the create form and submits with the chosen site", async () => {
    renderPage();
    fireEvent.click(await screen.findByRole("button", { name: /add asset/i }));
    const dialog = screen.getByRole("dialog");
    fireEvent.submit(dialog.querySelector("form") as HTMLFormElement);
    await waitFor(() => expect(screen.getAllByText("Required").length).toBeGreaterThan(0));
    expect(api.createAsset as ReturnType<typeof vi.fn>).not.toHaveBeenCalled();

    fireEvent.change(within(dialog).getByLabelText(/tag/i), { target: { value: "P-306" } });
    fireEvent.change(within(dialog).getByLabelText(/name/i), { target: { value: "Cooling Fan" } });
    fireEvent.change(within(dialog).getByLabelText(/class/i), { target: { value: "FAN" } });
    fireEvent.change(within(dialog).getByLabelText(/site/i), { target: { value: "s2" } });
    fireEvent.submit(dialog.querySelector("form") as HTMLFormElement);
    await waitFor(() =>
      expect(api.createAsset as ReturnType<typeof vi.fn>).toHaveBeenCalledWith({
        site_id: "s2",
        tag: "P-306",
        name: "Cooling Fan",
        class: "FAN"
      }),
    );
    expect(screen.getByText("Asset created")).toBeTruthy();
  });
});

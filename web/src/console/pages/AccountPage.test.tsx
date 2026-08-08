import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { AccountPage } from "./AccountPage";
import { I18nProvider } from "../../i18n/I18nProvider";
import { ThemeProvider } from "../../theme/ThemeProvider";
import { PrincipalProvider } from "../state/PrincipalProvider";
import { PreviewProvider } from "../state/PreviewProvider";
import type { Principal } from "../../types";

vi.mock("../../api", () => ({
  api: {
    principal: () =>
      Promise.resolve({
        id: "p1",
        display_name: "Dev Manager",
        organization_id: "org-1",
        site_ids: ["s1", "s2"],
        roles: ["Manager"],
        permissions: ["handover:write", "handover:accept", "recommendation:review"],
      } satisfies Principal),
  },
}));

function renderAccount() {
  return render(
    <ThemeProvider>
      <I18nProvider>
        <PreviewProvider>
          <PrincipalProvider>
            <MemoryRouter initialEntries={["/account"]}>
              <AccountPage />
            </MemoryRouter>
          </PrincipalProvider>
        </PreviewProvider>
      </I18nProvider>
    </ThemeProvider>,
  );
}

describe("AccountPage", () => {
  it("shows identity, roles, and grouped permissions", async () => {
    renderAccount();
    expect(await screen.findByText("Dev Manager")).toBeTruthy();
    expect(screen.getAllByText("Manager").length).toBeGreaterThan(0);
    expect(screen.getByText("handover:accept")).toBeTruthy();
    expect(screen.queryByText("workflow:publish")).toBeNull();
  });

  it("starts and exits a role preview", async () => {
    renderAccount();
    await screen.findByText("Dev Manager");
    fireEvent.click(screen.getByRole("button", { name: "Technician" }));
    expect(screen.getByRole("button", { name: "Use my real role" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Use my real role" }));
    expect(screen.queryByRole("button", { name: "Use my real role" })).toBeNull();
  });
});

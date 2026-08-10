import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { AuthorizedRoute, NotAuthorizedPage } from "./AuthorizedRoute";
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
        display_name: "Tech One",
        organization_id: "o1",
        site_ids: ["s1"],
        roles: ["Technician"],
        permissions: ["execution:write"],
      } satisfies Principal),
  },
}));

function renderWithPrincipal(children: React.ReactNode) {
  return render(
    <ThemeProvider>
      <I18nProvider>
        <PreviewProvider>
          <PrincipalProvider>
            <MemoryRouter initialEntries={["/quality"]}>{children}</MemoryRouter>
          </PrincipalProvider>
        </PreviewProvider>
      </I18nProvider>
    </ThemeProvider>,
  );
}

describe("AuthorizedRoute", () => {
  it("renders children when the permission is held", async () => {
    renderWithPrincipal(
      <AuthorizedRoute anyOf={["execution:write"]}>
        <div>protected body</div>
      </AuthorizedRoute>,
    );
    expect(await screen.findByText("protected body")).toBeTruthy();
  });

  it("renders the lock screen when the permission is missing", async () => {
    renderWithPrincipal(
      <AuthorizedRoute anyOf={["recommendation:review"]}>
        <div>protected body</div>
      </AuthorizedRoute>,
    );
    expect(await screen.findByText("You do not have access to this area")).toBeTruthy();
    expect(screen.getByText("Technician")).toBeTruthy();
    expect(screen.getByText("recommendation:review")).toBeTruthy();
    expect(screen.queryByText("protected body")).toBeNull();
  });
});

describe("NotAuthorizedPage", () => {
  it("shows fallback text when the principal has no roles", () => {
    render(
      <ThemeProvider>
        <I18nProvider>
          <MemoryRouter>
            <NotAuthorizedPage missing={["report:write"]} />
          </MemoryRouter>
        </I18nProvider>
      </ThemeProvider>,
    );
    expect(screen.getByText("No roles assigned")).toBeTruthy();
  });
});

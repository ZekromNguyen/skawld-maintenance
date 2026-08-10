import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { GlobalBar } from "./GlobalBar";
import { I18nProvider } from "../../i18n/I18nProvider";
import { ThemeProvider } from "../../theme/ThemeProvider";
import { SiteProvider } from "../state/SiteContext";

vi.mock("../../api", () => ({ api: {} }));

function renderBar(permissions: string[] = [], roles: string[] = []) {
  const principal = {
    id: "p1",
    display_name: "Tester",
    organization_id: "o1",
    site_ids: ["s1"],
    roles,
    permissions,
  };
  return render(
    <ThemeProvider>
      <I18nProvider>
        <SiteProvider principal={principal}>
          <MemoryRouter initialEntries={["/"]}>
            <Routes>
              <Route path="*" element={<GlobalBar principal={principal} />} />
            </Routes>
          </MemoryRouter>
        </SiteProvider>
      </I18nProvider>
    </ThemeProvider>,
  );
}

describe("GlobalBar", () => {
  it("renders operator identity and search", () => {
    renderBar();
    expect(screen.getByText("Tester")).toBeTruthy();
    expect(screen.getByRole("button", { name: /Search commands/ })).toBeTruthy();
  });
  it("opens the command palette on Cmd+K", () => {
    renderBar();
    fireEvent.keyDown(window, { key: "k", metaKey: true });
    expect(screen.getByText("Commands")).toBeTruthy();
  });
  it("hides the Create menu without create permissions", () => {
    renderBar();
    expect(screen.queryByRole("button", { name: /Create/ })).toBeNull();
  });
  it("shows gated Create items for a supervisor", () => {
    renderBar(["incident:create", "asset:create"], ["Maintenance Supervisor"]);
    fireEvent.click(screen.getByRole("button", { name: /Create/ }));
    expect(screen.getByRole("menuitem", { name: "New incident" })).toBeTruthy();
    expect(screen.getByRole("menuitem", { name: "New asset" })).toBeTruthy();
  });
  it("opens the avatar menu with roles and profile", () => {
    renderBar([], ["Technician"]);
    fireEvent.click(screen.getByRole("button", { name: "Profile" }));
    expect(screen.getByText("Technician")).toBeTruthy();
    expect(screen.getByRole("menuitem", { name: "Profile" })).toBeTruthy();
    expect(screen.getByRole("menuitem", { name: "Sign out" })).toBeTruthy();
  });
});

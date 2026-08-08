import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { GlobalBar } from "./GlobalBar";
import { I18nProvider } from "../../i18n/I18nProvider";
import { ThemeProvider } from "../../theme/ThemeProvider";
import { SiteProvider } from "../state/SiteContext";

vi.mock("../../api", () => ({ api: {} }));

function renderBar() {
  return render(
    <ThemeProvider>
      <I18nProvider>
        <SiteProvider
          principal={{
            id: "p1",
            display_name: "Tester",
            organization_id: "o1",
            site_ids: ["s1"],
            permissions: [],
          }}
        >
          <MemoryRouter initialEntries={["/"]}>
            <Routes>
              <Route
                path="*"
                element={
                  <GlobalBar
                    principal={{
                      id: "p1",
                      display_name: "Tester",
                      organization_id: "o1",
                      site_ids: ["s1"],
                      permissions: [],
                    }}
                  />
                }
              />
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
});

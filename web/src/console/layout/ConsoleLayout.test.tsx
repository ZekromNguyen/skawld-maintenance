import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { ConsoleLayout } from "./ConsoleLayout";
import { I18nProvider } from "../../i18n/I18nProvider";
import { ThemeProvider } from "../../theme/ThemeProvider";

describe("ConsoleLayout", () => {
  it("renders sidebar brand and outlet content", () => {
    render(
      <ThemeProvider>
      <I18nProvider>
        <MemoryRouter initialEntries={["/incidents"]}>
          <Routes>
            <Route element={<ConsoleLayout />}>
              <Route path="/incidents" element={<div>incident page body</div>} />
            </Route>
          </Routes>
        </MemoryRouter>
      </I18nProvider>
      </ThemeProvider>,
    );
    expect(screen.getByText("Skawld")).toBeTruthy();
    expect(screen.getByText("incident page body")).toBeTruthy();
  });
});

import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ThemeProvider, useTheme } from "./ThemeProvider";

function Probe() {
  const { theme, toggleTheme } = useTheme();
  return (
    <button onClick={toggleTheme} aria-label={`theme:${theme}`}>
      {theme}
    </button>
  );
}

describe("ThemeProvider", () => {
  it("defaults to light", () => {
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    );
    expect(screen.getByRole("button", { name: "theme:light" })).toBeTruthy();
    expect(document.documentElement.getAttribute("data-theme")).toBe("light");
  });

  it("toggles and persists the theme", () => {
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    );
    fireEvent.click(screen.getByRole("button", { name: "theme:light" }));
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    expect(localStorage.getItem("skawld.theme")).toBe("dark");
    expect(screen.getByRole("button", { name: "theme:dark" })).toBeTruthy();
  });
});

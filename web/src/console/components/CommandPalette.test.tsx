import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { CommandPalette } from "./CommandPalette";
import { I18nProvider } from "../../i18n/I18nProvider";

describe("CommandPalette", () => {
  it("opens with Ctrl+K and filters commands", async () => {
    render(
      <I18nProvider>
        <MemoryRouter>
          <CommandPalette principal={{ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: ["incident:create"] }} />
        </MemoryRouter>
      </I18nProvider>,
    );
    fireEvent.keyDown(window, { key: "k", ctrlKey: true });
    const dialog = await screen.findByRole("dialog");
    expect(dialog.textContent).toContain("Operations overview");
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "incident" } });
    expect(screen.getAllByRole("option").length).toBeGreaterThanOrEqual(1);
  });

  it("opens with / when not typing in an input", async () => {
    render(
      <I18nProvider>
        <MemoryRouter>
          <CommandPalette principal={{ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: [] }} />
        </MemoryRouter>
      </I18nProvider>,
    );
    fireEvent.keyDown(window, { key: "/" });
    expect(await screen.findByRole("dialog")).toBeTruthy();
  });

  it("does not open with / while typing in an input", async () => {
    render(
      <I18nProvider>
        <MemoryRouter>
          <CommandPalette principal={{ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: [] }} />
        </MemoryRouter>
      </I18nProvider>,
    );
    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();
    fireEvent.keyDown(input, { key: "/" });
    expect(screen.queryByRole("dialog")).toBeNull();
    document.body.removeChild(input);
  });

  it("runs the highlighted command on Enter", async () => {
    render(
      <I18nProvider>
        <MemoryRouter initialEntries={["/quality"]}>
          <Routes>
            <Route path="/incidents" element={<div>incident page probe</div>} />
            <Route path="*" element={<div>other</div>} />
          </Routes>
          <CommandPalette principal={{ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: [] }} />
        </MemoryRouter>
      </I18nProvider>,
    );
    fireEvent.keyDown(window, { key: "k", ctrlKey: true });
    const dialog = await screen.findByRole("dialog");
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "Incident execution" } });
    const option = await waitFor(() => {
      const found = Array.from(dialog.querySelectorAll('[role="option"]')).find(
        (el) => el.textContent === "Incident execution",
      );
      expect(found).toBeTruthy();
      return found as HTMLElement;
    });
    fireEvent.click(option);
    expect(await screen.findByText("incident page probe")).toBeTruthy();
    expect(screen.queryByRole("dialog")).toBeNull();
  });
});

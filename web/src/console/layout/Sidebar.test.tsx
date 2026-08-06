import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { Sidebar } from "./Sidebar";
import { I18nProvider } from "../../i18n/I18nProvider";

function renderSidebar(entry: string) {
  return render(
    <I18nProvider>
      <MemoryRouter initialEntries={[entry]}>
        <Routes>
          <Route path="/incidents/:incidentId" element={<div>detail</div>} />
          <Route path="*" element={<div>page</div>} />
        </Routes>
        <Sidebar principal={{ id: "p1", display_name: "T", organization_id: "o1", site_ids: ["s1"], permissions: [] }} />
      </MemoryRouter>
    </I18nProvider>,
  );
}

describe("Sidebar", () => {
  it("renders brand and grouped navigation", () => {
    renderSidebar("/");
    expect(screen.getByText("Skawld")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Incident execution" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "Executions" })).toBeTruthy();
  });

  it("marks the parent item active on detail routes", () => {
    renderSidebar("/incidents/inc1");
    const incidents = screen.getByRole("link", { name: "Incident execution" });
    expect(incidents.getAttribute("aria-current")).toBe("page");
  });

  it("marks only the exact item active on list routes", () => {
    renderSidebar("/incidents");
    expect(screen.getByRole("link", { name: "Incident execution" }).getAttribute("aria-current")).toBe("page");
    expect(screen.getByRole("link", { name: "Executions" }).getAttribute("aria-current")).toBeNull();
  });
});

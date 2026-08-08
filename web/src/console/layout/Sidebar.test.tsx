import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { Sidebar } from "./Sidebar";
import { I18nProvider } from "../../i18n/I18nProvider";
import { ThemeProvider } from "../../theme/ThemeProvider";
import { SiteProvider } from "../state/SiteContext";

function renderSidebar(entry: string, sites: string[] = ["s1"]) {
  return render(
    <ThemeProvider>
      <I18nProvider>
        <SiteProvider principal={{ id: "p1", display_name: "T", organization_id: "o1", site_ids: sites, permissions: [] }}>
        <MemoryRouter initialEntries={[entry]}>
          <Routes>
            <Route path="/incidents/:incidentId" element={<div>detail</div>} />
            <Route path="*" element={<div>page</div>} />
          </Routes>
          <Sidebar principal={{ id: "p1", display_name: "T", organization_id: "o1", site_ids: sites, permissions: [] }} />
        </MemoryRouter>
        </SiteProvider>
      </I18nProvider>
    </ThemeProvider>,
  );
}

function renderSidebarWithPermissions(permissions: string[], sites: string[] = ["s1"]) {
  return render(
    <ThemeProvider>
      <I18nProvider>
        <SiteProvider principal={{ id: "p1", display_name: "T", organization_id: "o1", site_ids: sites, permissions }}>
        <MemoryRouter initialEntries={["/"]}>
          <Routes>
            <Route path="*" element={<div>page</div>} />
          </Routes>
          <Sidebar
            principal={{
              id: "p1",
              display_name: "T",
              organization_id: "o1",
              site_ids: sites,
              permissions,
            }}
          />
        </MemoryRouter>
        </SiteProvider>
      </I18nProvider>
    </ThemeProvider>,
  );
}

describe("Sidebar", () => {
  it("renders brand and Jira-style grouped navigation", () => {
    renderSidebar("/");
    expect(screen.getByText("Skawld")).toBeTruthy();
    expect(screen.getByText("Primary")).toBeTruthy();
    expect(screen.getByText("Recommended")).toBeTruthy();
    expect(screen.getByText("Cross-links")).toBeTruthy();
    expect(screen.getByText("Customize")).toBeTruthy();
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

  it("shows Reports for a technician (report:write)", () => {
    renderSidebarWithPermissions(["report:write", "execution:write"]);
    expect(screen.getByRole("link", { name: "Reports" })).toBeTruthy();
  });

  it("hides Reports for a manager (no report:write)", () => {
    renderSidebarWithPermissions(["handover:accept", "recommendation:review"]);
    expect(screen.queryByRole("link", { name: "Reports" })).toBeNull();
  });

  it("lists recent sites only when the principal has more than one", () => {
    renderSidebar("/", ["s1", "s2"]);
    expect(screen.getByText("Recents")).toBeTruthy();
    expect(screen.getByRole("button", { name: /s2/i })).toBeTruthy();
  });

  it("hides recent sites for a single-site principal", () => {
    renderSidebar("/", ["s1"]);
    expect(screen.queryByText("Recents")).toBeNull();
  });
});

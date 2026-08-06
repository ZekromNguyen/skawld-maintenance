import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { PageHeader } from "./PageHeader";
import { PageTrailProvider } from "./PageTrail";
import { I18nProvider } from "../../i18n/I18nProvider";
import { SiteProvider } from "../state/SiteContext";

describe("PageHeader", () => {
  it("renders title, breadcrumbs, and actions", () => {
    render(
      <I18nProvider>
        <SiteProvider>
          <MemoryRouter>
            <PageTrailProvider trail={[{ label: "Incidents", to: "/incidents" }, { label: "IN-1042" }]}>
              <PageHeader title="Incident IN-1042" actions={<button>Resolve</button>} />
            </PageTrailProvider>
          </MemoryRouter>
        </SiteProvider>
      </I18nProvider>,
    );
    expect(screen.getByRole("heading", { level: 1, name: "Incident IN-1042" })).toBeTruthy();
    expect(screen.getByText("Incidents")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Resolve" })).toBeTruthy();
  });
});

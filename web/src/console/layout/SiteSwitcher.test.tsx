import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SiteSwitcher } from "./SiteSwitcher";
import { I18nProvider } from "../../i18n/I18nProvider";

describe("SiteSwitcher", () => {
  it("renders nothing for a single-site principal", () => {
    const { container } = render(
      <I18nProvider>
        <SiteSwitcher siteIds={["s1"]} value="s1" onChange={() => {}} />
      </I18nProvider>,
    );
    expect(container.firstChild).toBeNull();
  });

  it("switches site for multi-site principals", () => {
    let picked = "s1";
    render(
      <I18nProvider>
        <SiteSwitcher siteIds={["s1", "s2"]} value={picked} onChange={(id) => { picked = id; }} />
      </I18nProvider>,
    );
    fireEvent.change(screen.getByLabelText(/site/i), { target: { value: "s2" } });
    expect(picked).toBe("s2");
  });
});

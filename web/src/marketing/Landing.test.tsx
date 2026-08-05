import { describe, it, expect } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { I18nProvider } from "../i18n/I18nProvider";
import { Landing } from "./Landing";

function renderLanding() {
  return render(
    <I18nProvider>
      <Landing />
    </I18nProvider>
  );
}

describe("Landing marketing page", () => {
  it("renders hero value proposition and primary CTA in English by default", () => {
    renderLanding();
    expect(
      screen.getByRole("heading", {
        level: 1,
        name: /capture how your best technicians work/i,
      }),
    ).toBeTruthy();
    expect(screen.getAllByText("Book a pilot").length).toBeGreaterThan(0);
  });

  it("renders all major sections", () => {
    renderLanding();
    const sections = [
      "features",
      "workflow",
      "integrations",
      "security",
      "use-cases",
      "testimonials",
      "faq",
    ];
    for (const id of sections) {
      expect(document.getElementById(id)).toBeTruthy();
    }
  });

  it("renders FAQ answers when opened", () => {
    renderLanding();
    // StrictMode double-renders in test env, so the button may appear twice.
    const firstQuestion = screen.getAllByRole("button", {
      name: /what does a skawld pilot actually involve/i,
    })[0];
    firstQuestion.click();
    const answers = screen.getAllByText(
      /a bounded scope: one site or one pump line/i,
    );
    expect(answers.length).toBeGreaterThan(0);
  });

  it("switches the whole page to Vietnamese and back", () => {
    renderLanding();
    const select = screen.getByLabelText(/language/i) as HTMLSelectElement;
    fireEvent.change(select, { target: { value: "vi" } });
    expect(
      screen.getByRole("heading", {
        level: 1,
        name: /ghi lại cách những kỹ thuật viên giỏi nhất của bạn làm việc/i,
      }),
    ).toBeTruthy();
    expect(screen.getAllByText("Đặt lịch thí điểm").length).toBeGreaterThan(0);

    fireEvent.change(select, { target: { value: "en" } });
    expect(
      screen.getByRole("heading", {
        level: 1,
        name: /capture how your best technicians work/i,
      }),
    ).toBeTruthy();
  });

  it("translates FAQ content when Vietnamese is selected", () => {
    renderLanding();
    const select = screen.getByLabelText(/language/i) as HTMLSelectElement;
    fireEvent.change(select, { target: { value: "vi" } });
    const firstQuestion = screen.getAllByRole("button", {
      name: /thí điểm skawld thực sự bao gồm những gì/i,
    })[0];
    firstQuestion.click();
    const answers = screen.getAllByText(
      /một phạm vi giới hạn: một địa điểm hoặc một dây bơm/i,
    );
    expect(answers.length).toBeGreaterThan(0);
    fireEvent.change(select, { target: { value: "en" } });
  });

  it("shows exactly one P-302 tag in the hero panel header", () => {
    renderLanding();
    const select = screen.getByRole("combobox") as HTMLSelectElement;
    fireEvent.change(select, { target: { value: "en" } });
    expect(document.body.textContent).not.toContain("P-302 · P-302");
    expect(screen.getByText("Work order")).toBeTruthy();
    expect(screen.getByText("P-302 · Circulation pump inspection")).toBeTruthy();
  });

  it("renders the procedure meter with exactly one current phase", () => {
    renderLanding();
    const current = document.querySelectorAll('.hero-meter [data-current="true"]');
    expect(current.length).toBe(1);
    expect((current[0] as HTMLElement).style.background).toBe("var(--amber)");
  });

  it("renders integrations as self-contained monogram tiles", () => {
    renderLanding();
    const integrations = document.getElementById("integrations");
    expect(integrations).toBeTruthy();
    expect(integrations!.querySelectorAll("img").length).toBe(0);
    expect(integrations!.querySelectorAll("a").length).toBe(6);
    expect(integrations!.textContent).toContain("Slack");
  });

  it("opens the mobile menu and exposes nav links and sign-in", () => {
    renderLanding();
    const menuButton = screen.getByRole("button", { name: "Menu" });
    fireEvent.click(menuButton);
    const dialog = screen.getByRole("dialog");
    expect(dialog.textContent).toContain("Features");
    expect(dialog.textContent).toContain("Sign in");
  });

  it("keeps exactly one language switcher on the page", () => {
    renderLanding();
    expect(screen.getAllByLabelText(/language/i).length).toBe(1);
  });

  it("keeps card titles visibly above body text across sections", () => {
    renderLanding();
    const featuresTitles = Array.from(document.querySelectorAll("#features h3"));
    expect(featuresTitles.length).toBe(5);
    for (const h3 of featuresTitles) {
      expect((h3 as HTMLElement).style.fontSize).toBe("18px");
    }
    const workflowTitle = document.querySelector("#workflow h3") as HTMLElement;
    expect(workflowTitle.style.fontSize).toBe("20px");
  });
});

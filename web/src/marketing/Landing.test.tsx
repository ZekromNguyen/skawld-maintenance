import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Landing } from "./Landing";

describe("Landing marketing page", () => {
  it("renders hero value proposition and primary CTA", () => {
    render(<Landing />);
    expect(
      screen.getByRole("heading", {
        level: 1,
        name: /capture how your best technicians work/i,
      }),
    ).toBeTruthy();
    expect(screen.getAllByText("Book a pilot").length).toBeGreaterThan(0);
  });

  it("renders all major sections", () => {
    render(<Landing />);
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
    render(<Landing />);
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
});

import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { AnnunciatorStrip } from "./AnnunciatorStrip";

const cells = [
  {
    key: "open",
    label: "Open incidents",
    value: "12",
    tone: "critical" as const,
    to: "/incidents",
  },
  { key: "exec", label: "In progress", value: "3", tone: "medium" as const },
];

describe("AnnunciatorStrip", () => {
  it("renders a cell per item with tabular numerals", () => {
    render(
      <MemoryRouter>
        <AnnunciatorStrip cells={cells} />
      </MemoryRouter>,
    );
    expect(screen.getByText("Open incidents")).toBeTruthy();
    expect(screen.getByText("12")).toBeTruthy();
  });
  it("wraps cells with a link when to is set", () => {
    render(
      <MemoryRouter>
        <AnnunciatorStrip cells={cells} />
      </MemoryRouter>,
    );
    const link = screen.getByRole("link", { name: /Open incidents/ });
    expect(link.getAttribute("href")).toBe("/incidents");
  });
  it("renders a truthful zero", () => {
    render(
      <MemoryRouter>
        <AnnunciatorStrip
          cells={[{ key: "z", label: "Pending", value: "0", tone: "info" }]}
        />
      </MemoryRouter>,
    );
    expect(screen.getByText("0")).toBeTruthy();
  });
});

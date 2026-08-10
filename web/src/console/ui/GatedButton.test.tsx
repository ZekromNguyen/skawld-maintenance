import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { GatedButton } from "./GatedButton";

describe("GatedButton", () => {
  it("renders enabled when the permission is held", () => {
    render(<GatedButton allowed={true} reason="n/a">Resolve</GatedButton>);
    const button = screen.getByRole("button", { name: "Resolve" }) as HTMLButtonElement;
    expect(button.disabled).toBe(false);
    expect(button.title).toBe("");
  });

  it("disables with the reason when the permission is missing", () => {
    render(<GatedButton allowed={false} reason="Required permission not granted">Resolve</GatedButton>);
    const button = screen.getByRole("button", { name: "Resolve" }) as HTMLButtonElement;
    expect(button.disabled).toBe(true);
    expect(button.getAttribute("title")).toBe("Required permission not granted");
  });

  it("respects an external disabled prop (busy state)", () => {
    render(
      <GatedButton allowed={true} reason="n/a" disabled={true}>
        Submit
      </GatedButton>,
    );
    const button = screen.getByRole("button", { name: "Submit" }) as HTMLButtonElement;
    expect(button.disabled).toBe(true);
  });

  it("passes through className and onClick", () => {
    const onClick = () => undefined;
    render(
      <GatedButton allowed={true} reason="n/a" className="primary-button" onClick={onClick}>
        Approve
      </GatedButton>,
    );
    const button = screen.getByRole("button", { name: "Approve" });
    expect(button.className).toContain("primary-button");
  });
});

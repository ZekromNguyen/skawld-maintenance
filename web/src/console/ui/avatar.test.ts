import { describe, it, expect } from "vitest";
import { initials, avatarColor } from "./avatar";

describe("avatar helpers", () => {
  it("builds initials from first and last name", () => {
    expect(initials("Tran Minh")).toBe("TM");
    expect(initials("Ngoc Lan")).toBe("NL");
    expect(initials("Cher")).toBe("C");
    expect(initials("  ")).toBe("?");
  });

  it("is deterministic per name", () => {
    expect(avatarColor("Tran Minh")).toBe(avatarColor("Tran Minh"));
  });

  it("varies across common names", () => {
    const colors = new Set(["Tran Minh", "Le Hung", "Ngoc Lan", "Dev Supervisor"].map(avatarColor));
    expect(colors.size).toBeGreaterThan(1);
  });
});

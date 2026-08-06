import { describe, it, expect } from "vitest";
import { hasPermission, can, focusRole } from "./permissions";
import type { Principal } from "../types";

const principal = (permissions: string[]): Principal => ({
  id: "p1",
  display_name: "Tester",
  organization_id: "o1",
  site_ids: ["s1"],
  permissions,
});

describe("hasPermission", () => {
  it("is false without a principal", () => {
    expect(hasPermission(undefined, "report:write")).toBe(false);
  });

  it("is true when the permission is held", () => {
    expect(hasPermission(principal(["report:write"]), "report:write")).toBe(true);
  });
});

describe("can", () => {
  it("requires all listed permissions", () => {
    const p = principal(["report:write"]);
    expect(can(p, "report:write")).toBe(true);
    expect(can(p, ["report:write", "report:approve"])).toBe(false);
    expect(can(p, ["report:write"])).toBe(true);
  });

  it("allows an undefined requirement (ungated item)", () => {
    expect(can(principal([]), undefined)).toBe(true);
  });
});

describe("focusRole", () => {
  it("maps each role's signature permission set to its focus", () => {
    const admin = ["workflow:publish", "incident:resolve", "execution:write", "report:approve"];
    const supervisor = ["incident:resolve", "report:approve", "execution:write"];
    const senior = ["execution:prerequisite:verify", "execution:write", "demonstration:review"];
    const technician = ["execution:write", "report:write"];
    const manager = ["handover:accept", "recommendation:review"];
    expect(focusRole(principal(admin))).toBe("admin");
    expect(focusRole(principal(supervisor))).toBe("supervisor");
    expect(focusRole(principal(senior))).toBe("senior");
    expect(focusRole(principal(technician))).toBe("technician");
    expect(focusRole(principal(manager))).toBe("manager");
  });

  it("falls back to technician for read-only principals", () => {
    expect(focusRole(principal(["incident:read"]))).toBe("technician");
    expect(focusRole(undefined)).toBe("technician");
  });
});

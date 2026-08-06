import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { PermissionGate } from "./PermissionGate";
import type { Principal } from "../../types";

const principal = (permissions: string[]): Principal => ({
  id: "p1",
  display_name: "Tester",
  organization_id: "o1",
  site_ids: ["s1"],
  permissions,
});

describe("PermissionGate", () => {
  it("renders children when the permission is held", () => {
    render(
      <PermissionGate principal={principal(["report:write"])} required="report:write">
        <button type="button">Draft report</button>
      </PermissionGate>,
    );
    expect(screen.getByRole("button", { name: "Draft report" })).toBeTruthy();
  });

  it("renders nothing when the permission is missing", () => {
    render(
      <PermissionGate principal={principal(["incident:read"])} required="report:write">
        <button type="button">Draft report</button>
      </PermissionGate>,
    );
    expect(screen.queryByRole("button", { name: "Draft report" })).toBeNull();
  });

  it("supports an AND list of permissions", () => {
    render(
      <PermissionGate
        principal={principal(["report:write", "report:approve"])}
        required={["report:write", "report:approve"]}
      >
        <span>Approve</span>
      </PermissionGate>,
    );
    expect(screen.getByText("Approve")).toBeTruthy();
  });
});

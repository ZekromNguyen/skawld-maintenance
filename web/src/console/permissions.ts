import type { Principal } from "../types";

/** Every permission the console UI checks. Mirrors authorization.go. */
export type PermissionKey =
  | "asset:create"
  | "asset:criticality:approve"
  | "demonstration:capture"
  | "demonstration:review"
  | "execution:read:all"
  | "execution:write"
  | "execution:prerequisite:verify"
  | "handover:accept"
  | "handover:write"
  | "incident:create"
  | "incident:resolve"
  | "integration:external:import"
  | "knowledge:approve"
  | "knowledge:write"
  | "organization:create"
  | "recommendation:review"
  | "recommendation:run"
  | "report:approve"
  | "report:write"
  | "workflow:publish"
  | "workflow:review";

export function hasPermission(
  principal: Pick<Principal, "permissions"> | undefined,
  permission: PermissionKey,
): boolean {
  return principal?.permissions.includes(permission) ?? false;
}

export function can(
  principal: Pick<Principal, "permissions"> | undefined,
  required: PermissionKey | PermissionKey[] | undefined,
): boolean {
  if (!required) return true;
  const list = Array.isArray(required) ? required : [required];
  return list.every((permission) => hasPermission(principal, permission));
}

export type RoleFocus =
  | "admin"
  | "supervisor"
  | "senior"
  | "technician"
  | "manager";

/** Permission-derived focus, never role-name checks. Precedence matters. */
export function focusRole(
  principal: Pick<Principal, "permissions"> | undefined,
): RoleFocus {
  const has = (permission: PermissionKey) => hasPermission(principal, permission);
  if (has("workflow:publish")) return "admin";
  if (has("incident:resolve")) return "supervisor";
  if (has("execution:prerequisite:verify")) return "senior";
  if (has("execution:write")) return "technician";
  if (has("handover:accept")) return "manager";
  return "technician";
}

import type { PermissionKey } from "./permissions";

export type RoleName =
  | "Administrator"
  | "Maintenance Supervisor"
  | "Senior Technician"
  | "Technician"
  | "Manager";

export const ROLE_NAMES: RoleName[] = [
  "Administrator",
  "Maintenance Supervisor",
  "Senior Technician",
  "Technician",
  "Manager",
];

const SHARED_WRITE: PermissionKey[] = [
  "execution:write",
  "handover:write",
  "recommendation:run",
  "report:write",
];

/** Mirror of PermissionsForRole in internal/identity/domain/authorization.go.
 * Display and preview only; the API enforces the real mapping. */
export const ROLE_PERMISSIONS: Record<RoleName, PermissionKey[]> = {
  Administrator: [
    ...SHARED_WRITE,
    "organization:create",
    "asset:create",
    "asset:criticality:approve",
    "incident:create",
    "incident:resolve",
    "execution:read:all",
    "execution:prerequisite:verify",
    "knowledge:write",
    "knowledge:approve",
    "recommendation:review",
    "handover:accept",
    "demonstration:capture",
    "demonstration:review",
    "workflow:review",
    "workflow:publish",
    "report:approve",
    "integration:external:import",
    "field:manage",
  ],
  "Maintenance Supervisor": [
    ...SHARED_WRITE,
    "asset:create",
    "asset:criticality:approve",
    "incident:create",
    "incident:resolve",
    "execution:read:all",
    "execution:prerequisite:verify",
    "knowledge:write",
    "knowledge:approve",
    "recommendation:review",
    "handover:accept",
    "demonstration:capture",
    "demonstration:review",
    "workflow:review",
    "report:approve",
    "integration:external:import",
  ],
  "Senior Technician": [
    "execution:write",
    "execution:prerequisite:verify",
    "incident:create",
    "recommendation:run",
    "report:write",
    "handover:write",
    "demonstration:capture",
    "demonstration:review",
    "workflow:review",
  ],
  Technician: [
    "execution:write",
    "recommendation:run",
    "report:write",
    "handover:write",
    "demonstration:capture",
  ],
  Manager: [
    "recommendation:run",
    "recommendation:review",
    "handover:write",
    "handover:accept",
    "demonstration:review",
  ],
};

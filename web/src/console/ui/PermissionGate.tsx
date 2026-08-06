import type { ReactNode } from "react";
import type { Principal } from "../../types";
import { can, type PermissionKey } from "../permissions";

/**
 * PermissionGate: renders children only when the principal holds every
 * required permission. Missing permission hides the subtree entirely.
 */
export function PermissionGate({
  principal,
  required,
  children,
}: {
  principal?: Pick<Principal, "permissions">;
  required: PermissionKey | PermissionKey[];
  children: ReactNode;
}) {
  return can(principal, required) ? <>{children}</> : null;
}

import { usePrincipalValue } from "./state/PrincipalProvider";
import type { Principal } from "../types";

/** usePrincipal: current session principal from the app-wide provider. */
export function usePrincipal(): { data: Principal | undefined } {
  return { data: usePrincipalValue() };
}

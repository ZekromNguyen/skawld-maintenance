import { useApi } from "./useApi";
import { api } from "../api";
import type { Principal } from "../types";

/** usePrincipal: current session principal via the existing /me endpoint. */
export function usePrincipal() {
  return useApi<Principal>(() => api.principal());
}

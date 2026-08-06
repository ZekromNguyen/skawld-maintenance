import { createContext, useContext, type ReactNode } from "react";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import type { Principal } from "../../types";

const PrincipalContext = createContext<Principal | undefined>(undefined);

export function PrincipalProvider({ children }: { children: ReactNode }) {
  const { data } = useQuery(() => api.principal());
  return <PrincipalContext.Provider value={data}>{children}</PrincipalContext.Provider>;
}

export function usePrincipalValue(): Principal | undefined {
  return useContext(PrincipalContext);
}

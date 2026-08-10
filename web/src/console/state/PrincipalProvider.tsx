import { createContext, useContext, type ReactNode } from "react";
import { api } from "../../api";
import { useQuery } from "../useQuery";
import type { Principal } from "../../types";
import { applyPreview, usePreview } from "./PreviewProvider";

const PrincipalContext = createContext<Principal | undefined>(undefined);

export function PrincipalProvider({ children }: { children: ReactNode }) {
  const { data } = useQuery(() => api.principal());
  const { previewRole } = usePreview();
  const principal = applyPreview(data, previewRole);
  return <PrincipalContext.Provider value={principal}>{children}</PrincipalContext.Provider>;
}

export function usePrincipalValue(): Principal | undefined {
  return useContext(PrincipalContext);
}

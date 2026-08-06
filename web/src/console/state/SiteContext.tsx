import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import type { Principal } from "../../types";

interface SiteValue {
  siteId: string | undefined;
  setSiteId: (id: string) => void;
}

const SiteContext = createContext<SiteValue | null>(null);

export function SiteProvider({
  principal,
  children,
}: {
  principal?: Principal;
  children: ReactNode;
}) {
  const [siteId, setSiteId] = useState<string | undefined>(undefined);

  useEffect(() => {
    if (!siteId && principal?.site_ids[0]) {
      setSiteId(principal.site_ids[0]);
    }
  }, [principal, siteId]);

  return (
    <SiteContext.Provider value={{ siteId, setSiteId }}>
      {children}
    </SiteContext.Provider>
  );
}

export function useSite(): SiteValue {
  const value = useContext(SiteContext);
  if (!value) throw new Error("useSite must be used within SiteProvider");
  return value;
}

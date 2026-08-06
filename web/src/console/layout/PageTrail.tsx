import { createContext, useContext, type ReactNode } from "react";
import type { TrailItem } from "./Breadcrumbs";

const TrailContext = createContext<TrailItem[]>([]);

export function PageTrailProvider({
  trail,
  children,
}: {
  trail: TrailItem[];
  children: ReactNode;
}) {
  return <TrailContext.Provider value={trail}>{children}</TrailContext.Provider>;
}

export function usePageTrail(): TrailItem[] {
  return useContext(TrailContext);
}

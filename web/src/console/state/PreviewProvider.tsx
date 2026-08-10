import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import type { Principal } from "../../types";
import { ROLE_PERMISSIONS, type RoleName } from "../rolePermissions";

const PREVIEW_KEY = "skawld.previewRole";

type PreviewContextValue = {
  previewRole: RoleName | null;
  setPreviewRole: (role: RoleName | null) => void;
};

const PreviewContext = createContext<PreviewContextValue>({
  previewRole: null,
  setPreviewRole: () => {},
});

export function PreviewProvider({ children }: { children: ReactNode }) {
  const [previewRole, setPreviewRoleState] = useState<RoleName | null>(() => {
    const stored = sessionStorage.getItem(PREVIEW_KEY);
    return stored && stored in ROLE_PERMISSIONS ? (stored as RoleName) : null;
  });
  const setPreviewRole = useCallback((role: RoleName | null) => {
    if (role) sessionStorage.setItem(PREVIEW_KEY, role);
    else sessionStorage.removeItem(PREVIEW_KEY);
    setPreviewRoleState(role);
  }, []);
  const value = useMemo(() => ({ previewRole, setPreviewRole }), [previewRole, setPreviewRole]);
  return <PreviewContext.Provider value={value}>{children}</PreviewContext.Provider>;
}

export function usePreview() {
  return useContext(PreviewContext);
}

/** Overlays the preview role's permission mirror onto the real principal. */
export function applyPreview(principal: Principal | undefined, previewRole: RoleName | null): Principal | undefined {
  if (!principal || !previewRole) return principal;
  return { ...principal, permissions: [...ROLE_PERMISSIONS[previewRole]] };
}

import type { ReactNode } from "react";
import { Link } from "react-router";
import { LockKey } from "@phosphor-icons/react";
import { useI18n } from "../../i18n/I18nProvider";
import { usePrincipal } from "../usePrincipal";
import { canAny, type PermissionKey } from "../permissions";

/**
 * NotAuthorizedPage: Jira-style lock screen naming the principal's roles and
 * the missing permission, with exits back to the overview and the profile.
 */
export function NotAuthorizedPage({ missing }: { missing: PermissionKey[] }) {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  return (
    <section className="not-authorized">
      <LockKey size={28} aria-hidden="true" className="not-authorized-icon" />
      <h1>{t("authz.title")}</h1>
      <p>{t("authz.body")}</p>
      <dl className="not-authorized-meta">
        <div>
          <dt>{t("authz.yourRoles")}</dt>
          <dd>
            {principal?.roles?.length
              ? principal.roles.map((role) => (
                  <span key={role} className="role-badge">
                    {role}
                  </span>
                ))
              : t("authz.noRoles")}
          </dd>
        </div>
        <div>
          <dt>{t("authz.missingPermission")}</dt>
          <dd className="mono">{missing.join(" | ")}</dd>
        </div>
      </dl>
      <div className="not-authorized-actions">
        <Link to="/" className="primary-button">
          {t("authz.goHome")}
        </Link>
        <Link to="/account" className="secondary-button">
          {t("authz.viewAccount")}
        </Link>
      </div>
    </section>
  );
}

/**
 * AuthorizedRoute: renders children only when the principal holds the
 * required permission set (any-of). Mirrors backend enforcement so
 * restricted roles get an explicit screen instead of a failing fetch.
 */
export function AuthorizedRoute({
  anyOf,
  children,
}: {
  anyOf: PermissionKey[];
  children: ReactNode;
}) {
  const { data: principal } = usePrincipal();
  if (!principal) {
    return <div className="skeleton" style={{ height: 160 }} />;
  }
  if (!canAny(principal, anyOf)) {
    return <NotAuthorizedPage missing={anyOf} />;
  }
  return <>{children}</>;
}

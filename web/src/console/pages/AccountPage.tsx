import { useI18n } from "../../i18n/I18nProvider";
import type { MessageKey } from "../../i18n/messages";
import { usePrincipal } from "../usePrincipal";
import { usePreview } from "../state/PreviewProvider";
import { ROLE_NAMES, type RoleName } from "../rolePermissions";
import type { PermissionKey } from "../permissions";
import { PageHeader } from "../layout/PageHeader";

const GROUPS: { key: MessageKey; permissions: PermissionKey[] }[] = [
  { key: "account.group.assets", permissions: ["asset:create", "asset:criticality:approve"] },
  { key: "account.group.incidents", permissions: ["incident:create", "incident:resolve"] },
  {
    key: "account.group.executions",
    permissions: ["execution:write", "execution:read:all", "execution:prerequisite:verify"],
  },
  { key: "account.group.knowledge", permissions: ["knowledge:write", "knowledge:approve"] },
  { key: "account.group.reports", permissions: ["report:write", "report:approve"] },
  { key: "account.group.handovers", permissions: ["handover:write", "handover:accept"] },
  {
    key: "account.group.demonstrations",
    permissions: ["demonstration:capture", "demonstration:review"],
  },
  { key: "account.group.workflows", permissions: ["workflow:review", "workflow:publish"] },
  {
    key: "account.group.governance",
    permissions: ["organization:create", "recommendation:review", "integration:external:import"],
  },
];

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  const first = parts[0][0] ?? "?";
  const last = parts.length > 1 ? parts[parts.length - 1][0] ?? "" : "";
  return (first + last).toUpperCase();
}

/**
 * AccountPage: identity card (avatar, roles, organization, sites), the
 * principal's real permissions grouped by area, and the demo role preview
 * switcher.
 */
export function AccountPage() {
  const { t } = useI18n();
  const { data: principal } = usePrincipal();
  const { previewRole, setPreviewRole } = usePreview();
  const held = new Set(principal?.permissions ?? []);

  return (
    <>
      <PageHeader title={t("account.title")} />
      <section className="account-card">
        <span className="avatar-button avatar-button-large" aria-hidden="true">
          {initials(principal?.display_name ?? "")}
        </span>
        <div className="account-identity">
          <h2>{principal?.display_name ?? t("topbar.connecting")}</h2>
          <dl className="account-meta">
            <div>
              <dt>{t("account.roles")}</dt>
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
              <dt>{t("account.organization")}</dt>
              <dd className="mono">{principal?.organization_id ?? ""}</dd>
            </div>
            <div>
              <dt>{t("account.sites")}</dt>
              <dd className="mono">{(principal?.site_ids ?? []).join(", ")}</dd>
            </div>
          </dl>
        </div>
      </section>

      <section className="account-section">
        <h2>{t("account.permissions")}</h2>
        {GROUPS.map((group) => {
          const owned = group.permissions.filter((permission) => held.has(permission));
          if (owned.length === 0) return null;
          return (
            <div key={group.key} className="account-group">
              <span className="nav-section">{t(group.key)}</span>
              <div className="chip-row">
                {owned.map((permission) => (
                  <span key={permission} className="perm-chip mono">
                    {permission}
                  </span>
                ))}
              </div>
            </div>
          );
        })}
        {held.size === 0 ? <p>{t("account.noPermissions")}</p> : null}
      </section>

      <section className="account-section">
        <h2>{t("account.preview")}</h2>
        <p className="account-hint">{t("account.previewHint")}</p>
        <div className="chip-row">
          {ROLE_NAMES.map((role: RoleName) => (
            <button
              key={role}
              type="button"
              className={`role-badge role-badge-button${previewRole === role ? " active" : ""}`}
              aria-pressed={previewRole === role}
              onClick={() => setPreviewRole(role)}
            >
              {role}
            </button>
          ))}
          {previewRole ? (
            <button type="button" className="secondary-button" onClick={() => setPreviewRole(null)}>
              {t("account.exitPreview")}
            </button>
          ) : null}
        </div>
      </section>
    </>
  );
}

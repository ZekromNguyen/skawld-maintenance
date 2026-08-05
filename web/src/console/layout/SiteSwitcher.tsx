import { useI18n } from "../../i18n/I18nProvider";

export function SiteSwitcher({
  siteIds,
  value,
  onChange,
}: {
  siteIds: string[];
  value?: string;
  onChange: (siteId: string) => void;
}) {
  const { t } = useI18n();
  if (siteIds.length < 2) return null;
  return (
    <label className="site-switcher">
      <span className="eyebrow">{t("site.switcherLabel")}</span>
      <select value={value ?? ""} onChange={(e) => onChange(e.target.value)}>
        {siteIds.map((id) => (
          <option key={id} value={id}>
            {id}
          </option>
        ))}
      </select>
    </label>
  );
}

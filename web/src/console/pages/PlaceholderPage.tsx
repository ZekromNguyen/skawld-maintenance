import { useI18n } from "../../i18n/I18nProvider";
import type { MessageKey } from "../../i18n/messages";
import { Topbar } from "../layout/Topbar";

/**
 * PlaceholderPage: temporary route target used while pages are migrated.
 * Replaced by real pages in Tasks 3-10.
 */
export function PlaceholderPage({ titleKey }: { titleKey: MessageKey }) {
  const { t } = useI18n();
  return (
    <section>
      <Topbar title={t(titleKey)} />
      <div className="empty" style={{ padding: 40 }}>
        {t("placeholder.underConstruction")}
      </div>
    </section>
  );
}

import { useI18n } from "../../i18n/I18nProvider";
import type { MessageKey } from "../../i18n/messages";


export function EmptyRow({ columns, labelKey }: { columns: number; labelKey: MessageKey }) {
  const { t } = useI18n();
  return <tr><td className="empty" colSpan={columns}>{t(labelKey)}</td></tr>;
}

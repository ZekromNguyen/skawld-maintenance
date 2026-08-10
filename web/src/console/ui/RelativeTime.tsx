import { relativeTime } from "../../presentation";
import type { Locale } from "../../i18n/messages";

export function RelativeTime({ time, locale }: { time: string; locale: Locale }) {
  return <time dateTime={time}>{relativeTime(time, locale)}</time>;
}

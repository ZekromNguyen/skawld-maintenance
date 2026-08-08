import { relativeTime } from "../../presentation";
import type { Locale } from "../../i18n/messages";

export function RelativeTime({ time, locale }: { time: string; locale: Locale }) {
  const absolute = new Date(time).toLocaleString(locale === "vi" ? "vi-VN" : "en-US");
  return (
    <time dateTime={time} title={absolute}>
      {relativeTime(time, locale)}
    </time>
  );
}

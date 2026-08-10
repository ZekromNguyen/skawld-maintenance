import { Locale } from "./i18n/messages";

export function severityTone(severity: string): string {
  switch (severity) {
    case "CRITICAL":
      return "critical";
    case "HIGH":
      return "high";
    case "MEDIUM":
      return "medium";
    default:
      return "low";
  }
}

export function relativeTime(
  value: string,
  locale: Locale = "en",
  now = Date.now()
): string {
  const minutes = Math.max(
    0,
    Math.floor((now - new Date(value).getTime()) / 60_000)
  );
  if (minutes < 60) return locale === "vi" ? `${minutes} phút trước` : `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return locale === "vi" ? `${hours} giờ trước` : `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return locale === "vi" ? `${days} ngày trước` : `${days}d ago`;
}

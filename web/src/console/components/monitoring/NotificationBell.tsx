import { useEffect, useRef, useState } from "react";
import { Bell, BellRinging, X } from "@phosphor-icons/react";
import { useI18n } from "../../../i18n/I18nProvider";
import { api } from "../../../api";
import { StatusBadge, type Tone } from "../../ui/StatusBadge";
import type { MonitoringSummaryEntry } from "../../../types";

const POLL_MS = 60_000;

const STATUS_TONE: Record<MonitoringSummaryEntry["status"], Tone> = {
  PASS: "success",
  WARN: "high",
  CRIT: "critical",
};

const STATUS_LABEL_KEY: Record<MonitoringSummaryEntry["status"], string> = {
  PASS: "monitoring.statusPass",
  WARN: "monitoring.statusWarn",
  CRIT: "monitoring.statusCrit",
};

/**
 * NotificationBell: topbar indicator of open monitor alerts. Polls the
 * monitoring summary every 60s while mounted and lists non-PASS entries in a
 * dropdown. Rendered only for principals with monitoring:read.
 */
export function NotificationBell() {
  const { t } = useI18n();
  const [entries, setEntries] = useState<MonitoringSummaryEntry[]>([]);
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    const refresh = async () => {
      try {
        const page = await api.monitoringSummary();
        if (!cancelled) setEntries(page.items);
      } catch {
        // poll failures are silent; the page itself surfaces errors
      }
    };
    void refresh();
    const interval = window.setInterval(() => void refresh(), POLL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(interval);
    };
  }, []);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", onPointerDown);
    return () => document.removeEventListener("mousedown", onPointerDown);
  }, [open]);

  const active = entries.filter((entry) => entry.status !== "PASS");
  const count = active.reduce((sum, entry) => sum + entry.open_alerts, 0);

  return (
    <div className="notification-bell" ref={rootRef}>
      <button
        type="button"
        className="bell-button"
        aria-label={t("monitoring.alerts")}
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
      >
        {count > 0 ? <BellRinging size={18} aria-hidden="true" /> : <Bell size={18} aria-hidden="true" />}
        {count > 0 ? <span className="bell-count">{count > 99 ? "99+" : count}</span> : null}
      </button>
      {open ? (
        <div className="bell-dropdown" role="menu">
          <div className="bell-dropdown-head">
            <span className="eyebrow">{t("monitoring.alerts")}</span>
            <button
              type="button"
              className="bell-close"
              aria-label={t("monitoring.cancel")}
              onClick={() => setOpen(false)}
            >
              <X size={14} aria-hidden="true" />
            </button>
          </div>
          {active.length > 0 ? (
            <ul className="bell-list">
              {active.map((entry) => (
                <li key={entry.metric_key}>
                  <StatusBadge
                    tone={STATUS_TONE[entry.status]}
                    label={t(STATUS_LABEL_KEY[entry.status] as never)}
                  />
                  <span className="mono">{entry.metric_key}</span>
                  <span className="bell-value">
                    {entry.value} {entry.unit}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="bell-empty">{t("monitoring.noAlerts")}</p>
          )}
        </div>
      ) : null}
    </div>
  );
}

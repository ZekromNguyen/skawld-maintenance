import type { MessageKey } from "../i18n/messages";
import type { Tone } from "./ui/StatusBadge";

/** Single source of truth for incident severity labels and tones. */
export function severityLabelKey(severity: string): MessageKey {
  switch (severity) {
    case "CRITICAL":
      return "incident.severity.critical";
    case "HIGH":
      return "incident.severity.high";
    case "MEDIUM":
      return "incident.severity.medium";
    case "LOW":
      return "incident.severity.low";
    default:
      return "incident.severity.unknown";
  }
}

export function severityTone(severity: string): Tone {
  switch (severity) {
    case "CRITICAL":
      return "critical";
    case "HIGH":
      return "high";
    case "MEDIUM":
      return "medium";
    case "LOW":
      return "low";
    default:
      return "info";
  }
}

/** Single source of truth for incident state labels and tones. */
export function incidentStateLabelKey(state: string): MessageKey {
  switch (state) {
    case "OPEN":
      return "incident.state.open";
    case "IN_PROGRESS":
      return "incident.state.inProgress";
    case "RESOLVED":
      return "incident.state.resolved";
    default:
      return "incident.state.unknown";
  }
}

export function incidentStateTone(state: string): Tone {
  switch (state) {
    case "OPEN":
      return "high";
    case "IN_PROGRESS":
      return "medium";
    case "RESOLVED":
      return "success";
    default:
      return "info";
  }
}

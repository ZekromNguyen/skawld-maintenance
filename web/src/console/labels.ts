import type { MessageKey } from "../i18n/messages";
import type { Tone } from "./ui/StatusBadge";

/** Single source of truth for incident priority labels and tones. */
export function priorityLabelKey(priority: string): MessageKey {
  switch (priority) {
    case "CRITICAL":
      return "incident.priority.critical";
    case "HIGH":
      return "incident.priority.high";
    case "MEDIUM":
      return "incident.priority.medium";
    case "LOW":
      return "incident.priority.low";
    default:
      return "incident.priority.unknown";
  }
}

export function priorityTone(priority: string): Tone {
  switch (priority) {
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

/** Single source of truth for incident status labels and tones. */
export function incidentStatusLabelKey(status: string): MessageKey {
  switch (status) {
    case "OPEN":
      return "incident.status.open";
    case "IN_PROGRESS":
      return "incident.status.inProgress";
    case "RESOLVED":
      return "incident.status.resolved";
    case "CLOSED":
      return "incident.status.closed";
    case "REOPENED":
      return "incident.status.reopened";
    default:
      return "incident.status.unknown";
  }
}

export function incidentStatusTone(status: string): Tone {
  switch (status) {
    case "OPEN":
      return "high";
    case "IN_PROGRESS":
      return "medium";
    case "RESOLVED":
      return "success";
    case "CLOSED":
      return "low";
    case "REOPENED":
      return "medium";
    default:
      return "info";
  }
}

/** Workbench step risk tone: only safety-significant levels get a badge. */
export function riskTone(level: string): Tone | null {
  switch (level) {
    case "SAFETY_SIGNIFICANT":
      return "critical";
    case "ADVISORY":
      return "medium";
    default:
      return null;
  }
}

export function stepStateTone(state: string): Tone {
  switch (state) {
    case "COMPLETED":
      return "success";
    case "BLOCKED":
      return "critical";
    default:
      return "info";
  }
}

/** Asset status tone + label key. */
export function assetStatusTone(status: string): Tone {
  switch (status) {
    case "OPERATIONAL":
      return "success";
    case "MAINTENANCE":
      return "medium";
    case "DECOMMISSIONED":
      return "info";
    default:
      return "info";
  }
}

export function assetStatusLabelKey(status: string): MessageKey | null {
  switch (status) {
    case "OPERATIONAL":
      return "assets.status.operational";
    case "DECOMMISSIONED":
      return "assets.status.decommissioned";
    case "MAINTENANCE":
      return "assets.status.maintenance";
    default:
      return null;
  }
}

/** Report lifecycle state tone + label key. */
export function reportStateTone(state: string): Tone {
  switch (state) {
    case "APPROVED":
      return "success";
    case "SUBMITTED":
      return "medium";
    default:
      return "info";
  }
}

export function reportStateLabelKey(state: string): MessageKey | null {
  switch (state) {
    case "DRAFT":
      return "report.state.draft";
    case "SUBMITTED":
      return "report.state.submitted";
    case "APPROVED":
      return "report.state.approved";
    default:
      return null;
  }
}

/** Document revision approval status tone + label key. */
export function approvalTone(status: string): Tone {
  switch (status) {
    case "APPROVED":
      return "success";
    case "DRAFT":
      return "info";
    case "REVIEW_REQUIRED":
      return "medium";
    case "SUPERSEDED":
    case "RETIRED":
      return "low";
    default:
      return "info";
  }
}

export function approvalLabelKey(status: string): MessageKey | null {
  switch (status) {
    case "DRAFT":
      return "document.approval.draft";
    case "REVIEW_REQUIRED":
      return "document.approval.reviewRequired";
    case "APPROVED":
      return "document.approval.approved";
    case "SUPERSEDED":
      return "document.approval.superseded";
    case "RETIRED":
      return "document.approval.retired";
    default:
      return null;
  }
}

/** Shift handover state tone. */
export function handoverStateTone(state: string): Tone {
  switch (state) {
    case "ACKNOWLEDGED":
      return "success";
    case "SUBMITTED":
      return "medium";
    default:
      return "info";
  }
}

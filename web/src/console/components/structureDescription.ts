/**
 * structureDescription: turns a free-text incident description into
 * structured fields the way a Jira "describe in natural language" flow
 * would. Deterministic, keyword-driven, no network calls.
 */

export interface StructuredIncident {
  summary: string;
  details: string;
  priority: string;
  assetId: string;
}

const PRIORITY_RULES: Array<[string[], string]> = [
  [["critical", "emergency", "down", "failed", "fire", "explosion", "leak", "safety", "immediate", "halted", "stopped working"], "CRITICAL"],
  [["high", "severe", "loud", "hot", "overheat", "vibration", "alarm", "urgent", "major", "broken", "burning"], "HIGH"],
  [["medium", "moderate", "warning", "unusual", "noise", "warm", "fluctuat", "intermittent"], "MEDIUM"],
  [["low", "minor", "slight", "cosmetic", "routine"], "LOW"],
];

const OBSERVATION_KEYWORDS = [
  "vibrat", "noise", "temperature", "pressure", "reading", "smoke", "smell",
  "alarm", "sound", "hot", "leak", "oil", "rpm", "psi", "celsius", "mm/s",
  "measured", "observed", "reading",
];

const IMPACT_KEYWORDS = [
  "stopped", "shutdown", "downtime", "production", "output", "unavailable",
  "can't run", "cannot run", "halted", "down", "lost",
];

const ACTION_KEYWORDS = [
  "called", "notified", "switched", "isolated", "replaced", "tightened",
  "checked", "requested", "reported", "shut", "closed", "started", "restarted",
  "contacted", "escalated",
];

function splitSentences(text: string): string[] {
  return text
    .split(/(?<=[.!?])\s+/)
    .map((part) => part.trim())
    .filter(Boolean);
}

function truncate(text: string, max: number): string {
  const trimmed = text.trim().replace(/[.!?]+$/, "");
  if (trimmed.length <= max) return trimmed;
  return `${trimmed.slice(0, max - 1).trimEnd()}…`;
}

function suggestPriority(text: string): string {
  const lower = text.toLowerCase();
  for (const [keywords, level] of PRIORITY_RULES) {
    if (keywords.some((keyword) => lower.includes(keyword))) return level;
  }
  return "";
}

function matchAsset(text: string, assets: Array<{ id: string; tag: string }>): string {
  const lower = text.toLowerCase();
  for (const asset of assets) {
    if (asset.tag && lower.includes(asset.tag.toLowerCase())) return asset.id;
  }
  return "";
}

export function structureDescription(
  text: string,
  assets: Array<{ id: string; tag: string }>,
  labels: { what: string; observations: string; impact: string; actions: string },
): StructuredIncident {
  const sentences = splitSentences(text);
  const lower = text.toLowerCase();

  const what = sentences.slice(0, 1).join(" ");
  const observations = sentences.filter(
    (sentence) =>
      sentence !== what &&
      OBSERVATION_KEYWORDS.some((keyword) => sentence.toLowerCase().includes(keyword)),
  );
  const impact = sentences.filter(
    (sentence) => IMPACT_KEYWORDS.some((keyword) => sentence.toLowerCase().includes(keyword)),
  );
  const actions = sentences.filter(
    (sentence) => ACTION_KEYWORDS.some((keyword) => sentence.toLowerCase().includes(keyword)),
  );
  const remaining = sentences.filter(
    (sentence) =>
      sentence !== what &&
      !observations.includes(sentence) &&
      !impact.includes(sentence) &&
      !actions.includes(sentence),
  );

  const details: string[] = [];
  details.push(`${labels.what}: ${what}`);
  if (observations.length > 0) {
    details.push(`${labels.observations}: ${observations.join(" ")}`);
  }
  if (impact.length > 0) {
    details.push(`${labels.impact}: ${impact.join(" ")}`);
  }
  if (actions.length > 0) {
    details.push(`${labels.actions}: ${actions.join(" ")}`);
  }
  if (remaining.length > 0) {
    details.push(remaining.join(" "));
  }

  return {
    summary: truncate(what, 90),
    details: details.join("\n"),
    priority: suggestPriority(lower),
    assetId: matchAsset(text, assets),
  };
}

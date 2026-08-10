/** Jira-style avatar helpers: deterministic initials and hue per name. */

const AVATAR_COLORS = [
  "#0052cc",
  "#00875a",
  "#b65c00",
  "#de350b",
  "#5243aa",
  "#974aa6",
  "#008da6",
  "#4a6785",
];

export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  const first = parts[0][0] ?? "?";
  const last = parts.length > 1 ? parts[parts.length - 1][0] ?? "" : "";
  return (first + last).toUpperCase();
}

export function avatarColor(name: string): string {
  let hash = 0;
  for (const ch of name) {
    hash = (hash * 31 + ch.charCodeAt(0)) >>> 0;
  }
  return AVATAR_COLORS[hash % AVATAR_COLORS.length];
}

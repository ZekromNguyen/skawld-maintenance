export type Tone = "critical" | "high" | "medium" | "low" | "info" | "success";

export function StatusBadge({ tone, label }: { tone: Tone; label: string }) {
  return <span className={`tone tone-${tone}`}>{label}</span>;
}

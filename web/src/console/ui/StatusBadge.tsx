export type Tone = "critical" | "high" | "medium" | "low" | "info" | "success";

export function StatusBadge({
  tone,
  label,
  dot = false,
}: {
  tone: Tone;
  label: string;
  dot?: boolean;
}) {
  return (
    <span className={`tone tone-${tone}`}>
      {dot ? <span className={`badge-dot dot-${tone}`} aria-hidden="true" /> : null}
      {label}
    </span>
  );
}

/**
 * MonoStat: mono-numeral stat block for security / reliability metrics.
 * Numbers are real repo claims or marked sample; never fake precision.
 */
export function MonoStat({
  value,
  label,
  detail,
}: {
  value: string;
  label: string;
  detail?: string;
}) {
  return (
    <div
      style={{
        border: "1px solid var(--line)",
        background: "var(--surface)",
        borderRadius: 16,
        padding: "28px 26px",
      }}
    >
      <div
        className="mono-stat"
        style={{
          fontSize: 34,
          fontWeight: 700,
          lineHeight: 1.1,
          color: "var(--brand)",
          letterSpacing: "-0.02em",
        }}
      >
        {value}
      </div>
      <div
        style={{
          marginTop: 12,
          color: "var(--ink)",
          fontWeight: 600,
          fontSize: 14,
        }}
      >
        {label}
      </div>
      {detail ? (
        <div
          style={{
            marginTop: 6,
            color: "var(--ink-muted)",
            fontSize: 13,
            lineHeight: 1.5,
          }}
        >
          {detail}
        </div>
      ) : null}
    </div>
  );
}

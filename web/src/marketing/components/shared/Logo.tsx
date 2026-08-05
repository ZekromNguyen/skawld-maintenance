/**
 * Logo: Skawld wordmark. Simple geometric mark, brand-consistent.
 * A square instrument tile with an "S" cut, plus the wordmark in Geist.
 */
export function Logo({ size = 34, className = "" }: { size?: number; className?: string }) {
  return (
    <span
      className={className}
      style={{ display: "inline-flex", alignItems: "center", gap: 11 }}
    >
      <span
        aria-hidden="true"
        style={{
          width: size,
          height: size,
          display: "grid",
          placeItems: "center",
          borderRadius: 8,
          border: "1px solid var(--brand-dim)",
          background: "var(--sunken)",
          color: "var(--brand)",
          fontFamily: '"Geist Mono Variable", monospace',
          fontWeight: 800,
          fontSize: size * 0.5,
          letterSpacing: "0.02em",
        }}
      >
        S
      </span>
      <span
        className="logo-wordmark"
        style={{
          fontWeight: 700,
          fontSize: 17,
          letterSpacing: "0.01em",
          color: "var(--ink)",
          lineHeight: 1,
        }}
      >
        skawld
        <span
          style={{
            display: "block",
            color: "var(--ink-muted)",
            fontFamily: '"Geist Mono Variable", monospace',
            fontSize: 9,
            fontWeight: 500,
            letterSpacing: "0.14em",
            textTransform: "uppercase",
            marginTop: 3,
          }}
        >
          maintenance intelligence
        </span>
      </span>
    </span>
  );
}

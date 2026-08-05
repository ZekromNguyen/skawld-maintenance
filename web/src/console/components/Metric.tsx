import type { ReactNode } from "react";


export function Metric(props: { label: string; value: number; detail: string; accent?: boolean; safe?: boolean }) {
  return (
    <article className={`metric ${props.accent ? "metric-accent" : ""} ${props.safe ? "metric-safe" : ""}`}>
      <span>{props.label}</span>
      <strong>{props.value.toString().padStart(2, "0")}</strong>
      <small>{props.detail}</small>
    </article>
  );
}

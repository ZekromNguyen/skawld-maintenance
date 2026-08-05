import { useState } from "react";
import { Plus, Minus } from "@phosphor-icons/react";
import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";

/**
 * Faq: accessible accordion (native buttons, aria-expanded/controls).
 * Native details/summary would be lighter; buttons give styling control
 * while keeping keyboard semantics.
 */
const FAQS = [
  {
    q: "What does a Skawld pilot actually involve?",
    a: "A bounded scope: one site or one pump line, your equipment, your people. Skawld captures real executions and handovers, your supervisors review demonstrations, and we measure how the evidence-backed guidance holds up in the field.",
  },
  {
    q: "Who owns the data?",
    a: "You do. Skawld is self-hostable: your PostgreSQL, your S3-compatible object storage, your identity provider. Evidence, documents, and models live in your data plane.",
  },
  {
    q: "Does Skawld ever take control of equipment?",
    a: "No. Recommendations are advisory and always require human confirmation. LOTO verification gates and supervisor sign-offs are part of the workflow, not optional extras.",
  },
  {
    q: "Can it work offline?",
    a: "Yes. The technician client is offline-first with local persistence, built for field work without connectivity. When the connection returns, captured data reconciles.",
  },
  {
    q: "What about safety acceptance?",
    a: "A pilot is declared production-ready only after agreed RPO/RTO, safety acceptance, and signed distribution gates are met. We treat that discipline as a feature, not a footnote.",
  },
  {
    q: "Does it integrate with our existing systems?",
    a: "Identity via any OIDC provider, evidence in S3-compatible storage, truth in PostgreSQL, plus a pinned SDK contract and an OpenAPI surface for your internal integration work.",
  },
];

export function Faq() {
  const [open, setOpen] = useState<number | null>(0);

  return (
    <Section id="faq" title="Frequently asked questions">
      <Reveal>
        <div
          style={{
            marginTop: 40,
            borderTop: "1px solid var(--line)",
            display: "grid",
            gap: 0,
          }}
        >
          {FAQS.map((faq, index) => {
            const expanded = open === index;
            return (
              <div key={faq.q} style={{ borderBottom: "1px solid var(--line)" }}>
                <button
                  type="button"
                  aria-expanded={expanded}
                  aria-controls={`faq-answer-${index}`}
                  id={`faq-question-${index}`}
                  onClick={() => setOpen(expanded ? null : index)}
                  style={{
                    appearance: "none",
                    border: "none",
                    background: "transparent",
                    width: "100%",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-between",
                    gap: 16,
                    padding: "22px 4px",
                    cursor: "pointer",
                    textAlign: "left",
                    color: "var(--ink)",
                    fontSize: 16,
                    fontWeight: 600,
                  }}
                >
                  <span>{faq.q}</span>
                  <span
                    aria-hidden="true"
                    style={{
                      color: "var(--brand)",
                      flexShrink: 0,
                      display: "grid",
                      placeItems: "center",
                    }}
                  >
                    {expanded ? <Minus size={18} weight="bold" /> : <Plus size={18} weight="bold" />}
                  </span>
                </button>
                <div
                  id={`faq-answer-${index}`}
                  role="region"
                  aria-labelledby={`faq-question-${index}`}
                  style={{
                    display: expanded ? "block" : "none",
                    padding: "0 4px 24px",
                    color: "var(--ink-soft)",
                    fontSize: 14.5,
                    lineHeight: 1.65,
                    maxWidth: "68ch",
                  }}
                >
                  {faq.a}
                </div>
              </div>
            );
          })}
        </div>
      </Reveal>
    </Section>
  );
}

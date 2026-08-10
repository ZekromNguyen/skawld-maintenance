import { useState } from "react";
import { Plus, Minus } from "@phosphor-icons/react";
import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";
import { useI18n } from "../../i18n/I18nProvider";
import type { MessageKey } from "../../i18n/messages";

/**
 * Faq: accessible accordion (native buttons, aria-expanded/controls).
 * Native details/summary would be lighter; buttons give styling control
 * while keeping keyboard semantics.
 */
const FAQS: Array<{ qKey: MessageKey; aKey: MessageKey }> = [
  { qKey: "landing.faq.1.q", aKey: "landing.faq.1.a" },
  { qKey: "landing.faq.2.q", aKey: "landing.faq.2.a" },
  { qKey: "landing.faq.3.q", aKey: "landing.faq.3.a" },
  { qKey: "landing.faq.4.q", aKey: "landing.faq.4.a" },
  { qKey: "landing.faq.5.q", aKey: "landing.faq.5.a" },
  { qKey: "landing.faq.6.q", aKey: "landing.faq.6.a" },
];

export function Faq() {
  const { t } = useI18n();
  const [open, setOpen] = useState<number | null>(0);

  return (
    <Section id="faq" title={t("landing.faq.title")}>
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
              <div key={faq.qKey} style={{ borderBottom: "1px solid var(--line)" }}>
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
                  <span>{t(faq.qKey)}</span>
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
                  {t(faq.aKey)}
                </div>
              </div>
            );
          })}
        </div>
      </Reveal>
    </Section>
  );
}

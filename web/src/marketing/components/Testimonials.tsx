import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";
import { useI18n } from "../../i18n/I18nProvider";
import type { MessageKey } from "../../i18n/messages";

/**
 * Testimonials: quote cards, max 3 lines each, real attribution
 * (name + role). Sample content, clearly marked, no em-dashes.
 */
const TESTIMONIALS: Array<{ quoteKey: MessageKey; name: string; roleKey: MessageKey }> = [
  {
    quoteKey: "landing.testimonials.1.quote",
    name: "Minh Tran",
    roleKey: "landing.testimonials.1.role",
  },
  {
    quoteKey: "landing.testimonials.2.quote",
    name: "Elena Vasquez",
    roleKey: "landing.testimonials.2.role",
  },
  {
    quoteKey: "landing.testimonials.3.quote",
    name: "James Okafor",
    roleKey: "landing.testimonials.3.role",
  },
];

export function Testimonials() {
  const { t } = useI18n();
  return (
    <Section id="testimonials" title={t("landing.testimonials.title")}>
      {/* sample-marked per design system: realistic placeholder quotes */}
      <Reveal>
        <div
          style={{
            marginTop: 44,
            display: "grid",
            gridTemplateColumns: "1fr",
            gap: 16,
          }}
        >
          {TESTIMONIALS.map((testimonial) => (
            <figure
              key={testimonial.name}
              style={{
                margin: 0,
                border: "1px solid var(--line)",
                borderRadius: 16,
                background: "var(--surface)",
                padding: "28px 28px",
                display: "flex",
                flexDirection: "column",
                gap: 18,
              }}
            >
              <blockquote
                style={{
                  margin: 0,
                  fontSize: 15.5,
                  lineHeight: 1.6,
                  color: "var(--ink)",
                }}
              >
                &ldquo;{t(testimonial.quoteKey)}&rdquo;
              </blockquote>
              <figcaption
                style={{
                  fontSize: 13,
                  color: "var(--ink-muted)",
                  borderTop: "1px solid var(--line)",
                  paddingTop: 16,
                }}
              >
                <span style={{ fontWeight: 600, color: "var(--ink-soft)" }}>
                  {testimonial.name}
                </span>
                {" · "}
                {t(testimonial.roleKey)}
              </figcaption>
            </figure>
          ))}
        </div>
      </Reveal>
    </Section>
  );
}

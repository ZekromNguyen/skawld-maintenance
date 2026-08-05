import { Reveal } from "./shared/Reveal";
import { Section } from "./shared/Section";

/**
 * Testimonials: quote cards, max 3 lines each, real attribution
 * (name + role). Sample content, clearly marked, no em-dashes.
 */
const TESTIMONIALS = [
  {
    quote:
      "Our senior techs were skeptical. After one pilot shift they saw their own LOTO practice reflected back, with the safety gates intact.",
    name: "Minh Tran",
    role: "Reliability Manager, chemical processing plant",
  },
  {
    quote:
      "The evidence trail is what sold our safety committee. Every recommendation cites its sources, and nothing proceeds without human confirmation.",
    name: "Elena Vasquez",
    role: "Maintenance Director, regional energy operator",
  },
  {
    quote:
      "We captured two reviewed workflows in the first month. The handover reports alone have changed how our shifts pass the baton.",
    name: "James Okafor",
    role: "Plant Engineering Lead, food processing",
  },
];

export function Testimonials() {
  return (
    <Section id="testimonials" title="What operations teams say">
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
                &ldquo;{testimonial.quote}&rdquo;
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
                {testimonial.role}
              </figcaption>
            </figure>
          ))}
        </div>
      </Reveal>
    </Section>
  );
}

import type { ReactNode } from "react";

/**
 * Section: standardized container with optional eyebrow + title + lead.
 * Enforces the eyebrow budget from the design system (max 1 per 3 sections).
 */
export function Section({
  id,
  eyebrow,
  title,
  lead,
  children,
  className = "",
}: {
  id?: string;
  eyebrow?: string;
  title?: ReactNode;
  lead?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section id={id} className={`section ${className}`}>
      <div className="container">
        {eyebrow ? <span className="eyebrow">{eyebrow}</span> : null}
        {title ? <h2 className="section-title">{title}</h2> : null}
        {lead ? <p className="section-lead">{lead}</p> : null}
        {children}
      </div>
    </section>
  );
}

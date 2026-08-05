import type { ReactNode } from "react";

type ButtonVariant = "primary" | "ghost";

export function Button({
  children,
  href,
  variant = "primary",
  onClick,
  className = "",
}: {
  children: ReactNode;
  href?: string;
  variant?: ButtonVariant;
  onClick?: () => void;
  className?: string;
}) {
  const classes = `btn ${variant === "primary" ? "btn-primary" : "btn-ghost"} ${className}`;
  if (href) {
    return (
      <a href={href} className={classes}>
        {children}
      </a>
    );
  }
  return (
    <button type="button" onClick={onClick} className={classes}>
      {children}
    </button>
  );
}

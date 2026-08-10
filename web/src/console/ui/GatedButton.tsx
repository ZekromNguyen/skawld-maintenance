import type { ButtonHTMLAttributes } from "react";

/**
 * GatedButton: an action button that stays visible when the principal lacks
 * the required permission, but is disabled with an explanatory reason.
 * Never a no-op enabled click; never a mysteriously missing action.
 */
export function GatedButton({
  allowed,
  reason,
  disabled,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  allowed: boolean;
  reason: string;
}) {
  return (
    <button
      {...props}
      disabled={disabled || !allowed}
      title={allowed ? undefined : reason}
    />
  );
}

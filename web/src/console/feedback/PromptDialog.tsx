import { useEffect, useState } from "react";
import { Dialog } from "./Dialog";
import { FormField } from "../ui/FormField";

/** PromptDialog: single-field text input dialog replacing window.prompt. */
export function PromptDialog({
  open,
  onOpenChange,
  title,
  label,
  defaultValue = "",
  confirmLabel,
  pending = false,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  label: string;
  defaultValue?: string;
  confirmLabel: string;
  pending?: boolean;
  onConfirm: (value: string) => void;
}) {
  const [value, setValue] = useState(defaultValue);

  useEffect(() => {
    if (open) setValue(defaultValue);
  }, [open, defaultValue]);

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      footer={
        <>
          <button className="secondary-button" onClick={() => onOpenChange(false)} disabled={pending}>
            Cancel
          </button>
          <button
            className="primary-button"
            disabled={pending || !value.trim()}
            onClick={() => onConfirm(value.trim())}
          >
            {confirmLabel}
          </button>
        </>
      }
    >
      <FormField label={label} htmlFor="prompt-value">
        <textarea
          id="prompt-value"
          rows={3}
          value={value}
          onChange={(event) => setValue(event.target.value)}
        />
      </FormField>
    </Dialog>
  );
}

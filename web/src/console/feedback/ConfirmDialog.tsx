import { Dialog } from "./Dialog";

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  message,
  confirmLabel,
  pending = false,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  message: string;
  confirmLabel: string;
  pending?: boolean;
  onConfirm: () => void;
}) {
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      footer={
        <>
          <button
            className="secondary-button"
            onClick={() => onOpenChange(false)}
            disabled={pending}
          >
            Cancel
          </button>
          <button
            className="primary-button"
            onClick={onConfirm}
            disabled={pending}
          >
            {confirmLabel}
          </button>
        </>
      }
    >
      <p className="dialog-message">{message}</p>
    </Dialog>
  );
}

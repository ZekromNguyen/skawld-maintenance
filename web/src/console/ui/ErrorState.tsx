export function ErrorState({
  message,
  onRetry,
  onBack,
}: {
  message: string;
  onRetry?: () => void;
  onBack?: () => void;
}) {
  return (
    <div className="error-state" role="alert">
      <strong>Something went wrong</strong>
      <p>{message}</p>
      <div className="error-actions">
        {onRetry ? (
          <button className="secondary-button" onClick={onRetry}>
            Retry
          </button>
        ) : null}
        {onBack ? (
          <button className="secondary-button" onClick={onBack}>
            Back
          </button>
        ) : null}
      </div>
    </div>
  );
}

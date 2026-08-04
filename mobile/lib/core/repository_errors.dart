/// Typed, locale-independent error codes thrown by the technician repository
/// and attachment helpers. The repository layer stays Flutter-free; the UI
/// maps a [RepositoryException.code] to a localized message via
/// `AppLocalizations`.
enum RepositoryErrorCode {
  notAssigned,
  notInProgress,
  prerequisiteUnverified,
  copilotOffline,
  copilotEmptyResponse,
  attachmentSizeOutOfRange,
  unsupportedAttachmentType,
}

/// Thrown by the repository / attachment helpers. Extends [StateError] so the
/// English [message] stays available for logs and for existing
/// `throwsStateError` test matchers. [params] carries values the UI needs to
/// build the localized phrase (for example the prerequisite code).
class RepositoryException extends StateError {
  RepositoryException(this.code, String message, [this.params])
    : super(message);

  final RepositoryErrorCode code;
  final Map<String, Object>? params;
}

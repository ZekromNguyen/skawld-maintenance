enum SyncDisposition { applied, conflict, rejected, retry }

class SyncPolicy {
  const SyncPolicy();

  SyncDisposition classifyStatus(int status) {
    if (status >= 200 && status < 300) return SyncDisposition.applied;
    if (status == 409 || status == 412) return SyncDisposition.conflict;
    if (status == 408 || status == 425 || status == 429 || status >= 500) {
      return SyncDisposition.retry;
    }
    return SyncDisposition.rejected;
  }

  Duration retryDelay(int attemptCount) {
    final capped = attemptCount.clamp(1, 8);
    return Duration(seconds: (1 << capped) * 5);
  }

  int compareOutbox(
    DateTime firstCreatedAt,
    String firstId,
    DateTime secondCreatedAt,
    String secondId,
  ) {
    final timeOrder = firstCreatedAt.compareTo(secondCreatedAt);
    if (timeOrder != 0) return timeOrder;
    return firstId.compareTo(secondId);
  }
}

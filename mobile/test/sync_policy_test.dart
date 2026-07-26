import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:skawld_maintenance_mobile/data/database.dart';
import 'package:skawld_maintenance_mobile/sync/sync_policy.dart';

void main() {
  const policy = SyncPolicy();

  test(
    'duplicate outbox operation and client event is rejected locally',
    () async {
      final database = MaintenanceDatabase.forTesting(NativeDatabase.memory());
      addTearDown(database.close);
      final now = DateTime.utc(2026, 7, 26);
      final first = OutboxEntriesCompanion.insert(
        id: 'outbox-1',
        clientEventId: 'event-1',
        idempotencyKey: 'idempotency-1',
        operation: 'RECORD_MEASUREMENT',
        path: '/measurements',
        payloadJson: '{}',
        nextAttemptAt: now,
        createdAtDevice: now,
      );
      await database.into(database.outboxEntries).insert(first);
      await expectLater(
        database
            .into(database.outboxEntries)
            .insert(
              OutboxEntriesCompanion.insert(
                id: 'outbox-2',
                clientEventId: 'event-1',
                idempotencyKey: 'idempotency-2',
                operation: 'RECORD_MEASUREMENT',
                path: '/measurements',
                payloadJson: '{}',
                nextAttemptAt: now,
                createdAtDevice: now,
              ),
            ),
        throwsA(anything),
      );
    },
  );

  test('reordered device writes have stable chronological and ID order', () {
    final later = DateTime.utc(2026, 7, 26, 10);
    final earlier = later.subtract(const Duration(minutes: 1));
    expect(policy.compareOutbox(later, 'b', earlier, 'z'), greaterThan(0));
    expect(policy.compareOutbox(later, 'a', later, 'b'), lessThan(0));
  });

  test('interrupted requests retry with bounded exponential backoff', () {
    expect(policy.classifyStatus(408), SyncDisposition.retry);
    expect(policy.classifyStatus(429), SyncDisposition.retry);
    expect(policy.classifyStatus(503), SyncDisposition.retry);
    expect(policy.retryDelay(1), const Duration(seconds: 10));
    expect(policy.retryDelay(100), const Duration(seconds: 1280));
  });

  test('stale versions become visible conflicts and do not retry silently', () {
    expect(policy.classifyStatus(409), SyncDisposition.conflict);
    expect(policy.classifyStatus(412), SyncDisposition.conflict);
    expect(policy.classifyStatus(422), SyncDisposition.rejected);
    expect(policy.classifyStatus(204), SyncDisposition.applied);
  });
}

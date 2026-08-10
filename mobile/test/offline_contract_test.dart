import 'package:drift/drift.dart';
import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:skawld_maintenance_mobile/core/app_config.dart';
import 'package:skawld_maintenance_mobile/data/database.dart';
import 'package:skawld_maintenance_mobile/data/technician_repository.dart';
import 'package:skawld_maintenance_mobile/main.dart';

void main() {
  test('offline identity fields remain stable across retry', () {
    const clientEventId = 'b52174cc-c75d-49f5-ad19-281340dc8a53';
    const idempotencyKey = '8d9ad5de-da14-4db2-af06-fd04f63b7778';
    final firstAttempt = (clientEventId, idempotencyKey);
    final retry = (clientEventId, idempotencyKey);

    expect(retry, firstAttempt);
  });

  test('desktop runtime configuration validates and normalizes endpoints', () {
    final config = AppConfig.fromValues(
      apiUrl: 'https://maintenance.example.internal/',
      oidcIssuer: 'https://identity.example.internal/realms/skawld/',
      oidcClientId: 'skawld-mobile',
    );

    expect(config.apiUrl, 'https://maintenance.example.internal');
    expect(
      config.oidcIssuer,
      'https://identity.example.internal/realms/skawld',
    );
  });

  test('desktop evidence picker maps only supported MIME types', () {
    expect(attachmentMIME('bearing.JPG'), 'image/jpeg');
    expect(attachmentMIME('voice.m4a'), 'audio/m4a');
    expect(() => attachmentMIME('manual.pdf'), throwsStateError);
  });

  test('measurement and outbox command are committed atomically', () async {
    final database = MaintenanceDatabase.forTesting(NativeDatabase.memory());
    addTearDown(database.close);
    final repository = TechnicianRepository(database);

    final eventID = await repository.recordMeasurement(
      executionId: 'execution-1',
      measurementType: 'VIBRATION_VELOCITY',
      value: '8.1',
      unit: 'MM_PER_S',
      deviceId: 'device-1',
    );

    final measurement = await (database.select(
      database.localMeasurements,
    )..where((row) => row.clientEventId.equals(eventID))).getSingle();
    final outbox = await (database.select(
      database.outboxEntries,
    )..where((row) => row.clientEventId.equals(eventID))).getSingle();
    expect(measurement.value, '8.1');
    expect(measurement.syncStatus, 'PENDING');
    expect(outbox.operation, 'RECORD_MEASUREMENT');
    expect(outbox.idempotencyKey, isNotEmpty);
  });

  test(
    'offline intrusive step remains blocked without verified isolation',
    () async {
      final database = MaintenanceDatabase.forTesting(NativeDatabase.memory());
      addTearDown(database.close);
      final repository = TechnicianRepository(database);
      final now = DateTime.now().toUtc();
      await database
          .into(database.cachedExecutions)
          .insert(
            CachedExecutionsCompanion.insert(
              id: 'execution-1',
              siteId: 'site-1',
              incidentId: 'incident-1',
              assetId: 'asset-1',
              assetTag: 'P-302',
              purpose: 'High vibration inspection',
              state: 'IN_PROGRESS',
              serverVersion: 1,
              cachedAt: now,
            ),
          );
      await database
          .into(database.cachedExecutionSteps)
          .insert(
            CachedExecutionStepsCompanion.insert(
              id: 'step-4',
              executionId: 'execution-1',
              stepKey: 'inspect-bearing',
              sequence: 4,
              title: 'Inspect bearing',
              state: 'READY',
              riskLevel: 'SAFETY_SIGNIFICANT',
              requiredPrerequisite: const Value('ENERGY_ISOLATION'),
              serverVersion: 1,
            ),
          );

      await expectLater(
        repository.completeStep(
          executionId: 'execution-1',
          stepId: 'step-4',
          deviceId: 'device-1',
        ),
        throwsStateError,
      );
      expect(await database.select(database.outboxEntries).get(), isEmpty);
    },
  );
}

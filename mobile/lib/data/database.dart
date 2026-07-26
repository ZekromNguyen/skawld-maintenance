import 'package:drift/drift.dart';
import 'package:drift_flutter/drift_flutter.dart';

part 'database.g.dart';

class CachedAssets extends Table {
  TextColumn get id => text()();
  TextColumn get siteId => text()();
  TextColumn get tag => text()();
  TextColumn get name => text()();
  TextColumn get assetClass => text()();
  TextColumn get sourceOfTruth => text()();
  TextColumn get criticality => text().nullable()();
  IntColumn get serverVersion => integer()();
  DateTimeColumn get cachedAt => dateTime()();

  @override
  Set<Column<Object>> get primaryKey => {id};
}

class CachedIncidents extends Table {
  TextColumn get id => text()();
  TextColumn get siteId => text()();
  TextColumn get assetId => text()();
  TextColumn get assetTag => text()();
  TextColumn get number => text()();
  TextColumn get summary => text()();
  TextColumn get severity => text()();
  TextColumn get state => text()();
  DateTimeColumn get detectedAt => dateTime()();
  IntColumn get serverVersion => integer()();
  DateTimeColumn get cachedAt => dateTime()();

  @override
  Set<Column<Object>> get primaryKey => {id};
}

class CachedExecutions extends Table {
  TextColumn get id => text()();
  TextColumn get siteId => text()();
  TextColumn get incidentId => text()();
  TextColumn get assetId => text()();
  TextColumn get assetTag => text()();
  TextColumn get purpose => text()();
  TextColumn get state => text()();
  IntColumn get serverVersion => integer()();
  TextColumn get verifiedPrerequisites =>
      text().withDefault(const Constant(''))();
  DateTimeColumn get cachedAt => dateTime()();

  @override
  Set<Column<Object>> get primaryKey => {id};
}

class CachedExecutionSteps extends Table {
  TextColumn get id => text()();
  TextColumn get executionId => text()();
  TextColumn get stepKey => text()();
  IntColumn get sequence => integer()();
  TextColumn get title => text()();
  TextColumn get state => text()();
  TextColumn get riskLevel => text()();
  TextColumn get requiredPrerequisite => text().nullable()();
  TextColumn get blockedReason => text().nullable()();
  IntColumn get serverVersion => integer()();

  @override
  Set<Column<Object>> get primaryKey => {id};
}

class LocalMeasurements extends Table {
  TextColumn get clientEventId => text()();
  TextColumn get executionId => text()();
  TextColumn get componentId => text().nullable()();
  TextColumn get measurementType => text()();
  TextColumn get value => text()();
  TextColumn get unit => text()();
  TextColumn get source => text().withDefault(const Constant('MANUAL'))();
  TextColumn get dataQuality => text().withDefault(const Constant('GOOD'))();
  TextColumn get verificationStatus =>
      text().withDefault(const Constant('UNVERIFIED'))();
  DateTimeColumn get observedAt => dateTime()();
  DateTimeColumn get createdAtDevice => dateTime()();
  TextColumn get syncStatus => text().withDefault(const Constant('PENDING'))();
  TextColumn get serverId => text().nullable()();
  TextColumn get syncError => text().nullable()();

  @override
  Set<Column<Object>> get primaryKey => {clientEventId};
}

class LocalObservations extends Table {
  TextColumn get clientEventId => text()();
  TextColumn get executionId => text()();
  TextColumn get componentId => text().nullable()();
  TextColumn get property => text().nullable()();
  TextColumn get status => text().nullable()();
  TextColumn get narrative => text()();
  TextColumn get source => text().withDefault(const Constant('TECHNICIAN'))();
  TextColumn get verificationStatus =>
      text().withDefault(const Constant('UNVERIFIED'))();
  DateTimeColumn get observedAt => dateTime()();
  DateTimeColumn get createdAtDevice => dateTime()();
  TextColumn get syncStatus => text().withDefault(const Constant('PENDING'))();
  TextColumn get serverId => text().nullable()();
  TextColumn get syncError => text().nullable()();

  @override
  Set<Column<Object>> get primaryKey => {clientEventId};
}

class LocalAttachments extends Table {
  TextColumn get clientEventId => text()();
  TextColumn get entityKind => text()();
  TextColumn get entityId => text()();
  TextColumn get siteId => text()();
  TextColumn get localPath => text()();
  TextColumn get filename => text()();
  TextColumn get mimeType => text()();
  IntColumn get sizeBytes => integer()();
  TextColumn get checksumSha256 => text()();
  TextColumn get syncStatus => text().withDefault(const Constant('PENDING'))();
  TextColumn get serverId => text().nullable()();
  TextColumn get syncError => text().nullable()();
  DateTimeColumn get createdAtDevice => dateTime()();

  @override
  Set<Column<Object>> get primaryKey => {clientEventId};
}

class OutboxEntries extends Table {
  TextColumn get id => text()();
  TextColumn get clientEventId => text()();
  TextColumn get idempotencyKey => text()();
  TextColumn get operation => text()();
  TextColumn get path => text()();
  TextColumn get payloadJson => text()();
  TextColumn get status => text().withDefault(const Constant('PENDING'))();
  IntColumn get attemptCount => integer().withDefault(const Constant(0))();
  DateTimeColumn get nextAttemptAt => dateTime()();
  DateTimeColumn get createdAtDevice => dateTime()();
  DateTimeColumn get lastAttemptAt => dateTime().nullable()();
  TextColumn get lastError => text().nullable()();
  IntColumn get baseServerVersion => integer().nullable()();

  @override
  Set<Column<Object>> get primaryKey => {id};

  @override
  List<Set<Column<Object>>> get uniqueKeys => [
    {operation, clientEventId},
  ];
}

@DriftDatabase(
  tables: [
    CachedAssets,
    CachedIncidents,
    CachedExecutions,
    CachedExecutionSteps,
    LocalMeasurements,
    LocalObservations,
    LocalAttachments,
    OutboxEntries,
  ],
)
class MaintenanceDatabase extends _$MaintenanceDatabase {
  MaintenanceDatabase()
    : super(
        driftDatabase(
          name: 'skawld-maintenance',
          native: const DriftNativeOptions(shareAcrossIsolates: true),
        ),
      );

  MaintenanceDatabase.forTesting(super.executor);

  @override
  int get schemaVersion => 1;

  Future<void> cacheExecution(Map<String, dynamic> value) async {
    final now = DateTime.now().toUtc();
    final verified = (value['prerequisites'] as List<dynamic>? ?? const [])
        .cast<Map<String, dynamic>>()
        .where((item) {
          if (item['status'] != 'VERIFIED') return false;
          final validUntil = item['valid_until'] as String?;
          return validUntil == null ||
              DateTime.parse(validUntil).toUtc().isAfter(now);
        })
        .map((item) => item['type'] as String)
        .toSet()
        .join(',');
    await transaction(() async {
      await into(cachedExecutions).insertOnConflictUpdate(
        CachedExecutionsCompanion.insert(
          id: value['id'] as String,
          siteId: value['site_id'] as String,
          incidentId: value['incident_id'] as String,
          assetId: value['asset_id'] as String,
          assetTag: value['asset_tag'] as String? ?? '',
          purpose: value['purpose'] as String,
          state: value['state'] as String,
          serverVersion: value['version'] as int,
          verifiedPrerequisites: Value(verified),
          cachedAt: now,
        ),
      );
      for (final raw in value['steps'] as List<dynamic>? ?? const []) {
        final step = raw as Map<String, dynamic>;
        await into(cachedExecutionSteps).insertOnConflictUpdate(
          CachedExecutionStepsCompanion.insert(
            id: step['id'] as String,
            executionId: value['id'] as String,
            stepKey: step['key'] as String,
            sequence: step['sequence'] as int,
            title: step['title'] as String,
            state: step['state'] as String,
            riskLevel: step['risk_level'] as String,
            requiredPrerequisite: Value(
              step['required_prerequisite'] as String?,
            ),
            blockedReason: Value(step['blocked_reason'] as String?),
            serverVersion: step['version'] as int,
          ),
        );
      }
    });
  }
}

import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:drift/drift.dart';
import 'package:uuid/uuid.dart';

import '../core/access_token.dart';
import 'database.dart';

class TechnicianRepository {
  TechnicianRepository(
    this.database, {
    Dio? http,
    AccessTokenProvider? tokens,
    Uuid? uuid,
  }) : _http = http,
       _tokens = tokens,
       _uuid = uuid ?? const Uuid();

  final MaintenanceDatabase database;
  final Dio? _http;
  final AccessTokenProvider? _tokens;
  final Uuid _uuid;

  Stream<List<CachedIncident>> watchActiveIncidents() {
    return (database.select(database.cachedIncidents)
          ..where((row) => row.state.isNotValue('RESOLVED'))
          ..orderBy([(row) => OrderingTerm.desc(row.detectedAt)]))
        .watch();
  }

  Stream<List<CachedExecution>> watchAssignedExecutions() {
    return (database.select(database.cachedExecutions)
          ..where((row) => row.state.isNotValue('COMPLETED'))
          ..orderBy([(row) => OrderingTerm.desc(row.cachedAt)]))
        .watch();
  }

  Stream<List<CachedExecutionStep>> watchSteps(String executionId) {
    return (database.select(database.cachedExecutionSteps)
          ..where((row) => row.executionId.equals(executionId))
          ..orderBy([(row) => OrderingTerm.asc(row.sequence)]))
        .watch();
  }

  Stream<List<OutboxEntry>> watchSyncIssues() {
    return (database.select(
      database.outboxEntries,
    )..where((row) => row.status.isIn(const ['CONFLICT', 'REJECTED']))).watch();
  }

  Future<void> startExecution(
    String executionId, {
    required String deviceId,
  }) async {
    await database.transaction(() async {
      final execution = await (database.select(
        database.cachedExecutions,
      )..where((row) => row.id.equals(executionId))).getSingle();
      if (execution.state != 'ASSIGNED') {
        throw StateError('Only an assigned execution can start');
      }
      final now = DateTime.now().toUtc();
      await (database.update(
        database.cachedExecutions,
      )..where((row) => row.id.equals(executionId))).write(
        CachedExecutionsCompanion(
          state: const Value('IN_PROGRESS'),
          serverVersion: Value(execution.serverVersion + 1),
        ),
      );
      final eventId = _uuid.v4();
      await database
          .into(database.outboxEntries)
          .insert(
            OutboxEntriesCompanion.insert(
              id: _uuid.v4(),
              clientEventId: eventId,
              idempotencyKey: _uuid.v4(),
              operation: 'START_EXECUTION',
              path: '/api/v1/executions/$executionId/start',
              payloadJson: jsonEncode({
                'expected_version': execution.serverVersion,
                'client_event_id': eventId,
                'device_id': deviceId,
                'created_at_device': now.toIso8601String(),
                'payload_version': 1,
                'base_server_version': execution.serverVersion,
              }),
              nextAttemptAt: now,
              createdAtDevice: now,
              baseServerVersion: Value(execution.serverVersion),
            ),
          );
    });
  }

  Future<String> recordMeasurement({
    required String executionId,
    String? componentId,
    required String measurementType,
    required String value,
    required String unit,
    required String deviceId,
    DateTime? observedAt,
  }) async {
    final clientEventId = _uuid.v4();
    final idempotencyKey = _uuid.v4();
    final createdAt = DateTime.now().toUtc();
    final payload = <String, Object?>{
      'client_event_id': clientEventId,
      'device_id': deviceId,
      'created_at_device': createdAt.toIso8601String(),
      'component_id': componentId,
      'measurement_type': measurementType,
      'value': value,
      'unit': unit,
      'source': 'MANUAL',
      'data_quality': 'GOOD',
      'verification_status': 'UNVERIFIED',
      'observed_at': (observedAt ?? createdAt).toUtc().toIso8601String(),
    };

    await database.transaction(() async {
      await database
          .into(database.localMeasurements)
          .insert(
            LocalMeasurementsCompanion.insert(
              clientEventId: clientEventId,
              executionId: executionId,
              componentId: Value(componentId),
              measurementType: measurementType,
              value: value,
              unit: unit,
              observedAt: observedAt ?? createdAt,
              createdAtDevice: createdAt,
            ),
          );
      await database
          .into(database.outboxEntries)
          .insert(
            OutboxEntriesCompanion.insert(
              id: _uuid.v4(),
              clientEventId: clientEventId,
              idempotencyKey: idempotencyKey,
              operation: 'RECORD_MEASUREMENT',
              path: '/api/v1/executions/$executionId/measurements',
              payloadJson: jsonEncode(payload),
              nextAttemptAt: createdAt,
              createdAtDevice: createdAt,
            ),
          );
    });
    return clientEventId;
  }

  Future<String> completeStep({
    required String executionId,
    required String stepId,
    required String deviceId,
  }) async {
    final clientEventId = _uuid.v4();
    await database.transaction(() async {
      final execution = await (database.select(
        database.cachedExecutions,
      )..where((row) => row.id.equals(executionId))).getSingle();
      final step =
          await (database.select(database.cachedExecutionSteps)..where(
                (row) =>
                    row.id.equals(stepId) & row.executionId.equals(executionId),
              ))
              .getSingle();
      if (execution.state != 'IN_PROGRESS') {
        throw StateError('Execution is not in progress');
      }
      final prerequisite = step.requiredPrerequisite;
      final verified = execution.verifiedPrerequisites
          .split(',')
          .where((item) => item.isNotEmpty)
          .toSet();
      if (prerequisite != null && !verified.contains(prerequisite)) {
        throw StateError(
          '$prerequisite must be verified by the external authority while online',
        );
      }
      if (step.state == 'COMPLETED') return;
      final now = DateTime.now().toUtc();
      await (database.update(
        database.cachedExecutionSteps,
      )..where((row) => row.id.equals(stepId))).write(
        CachedExecutionStepsCompanion(
          state: const Value('COMPLETED'),
          serverVersion: Value(step.serverVersion + 1),
          blockedReason: const Value(null),
        ),
      );
      await (database.update(
        database.cachedExecutions,
      )..where((row) => row.id.equals(executionId))).write(
        CachedExecutionsCompanion(
          serverVersion: Value(execution.serverVersion + 1),
        ),
      );
      await database
          .into(database.outboxEntries)
          .insert(
            OutboxEntriesCompanion.insert(
              id: _uuid.v4(),
              clientEventId: clientEventId,
              idempotencyKey: _uuid.v4(),
              operation: 'COMPLETE_STEP',
              path: '/api/v1/executions/$executionId/steps/$stepId/complete',
              payloadJson: jsonEncode({
                'expected_execution_version': execution.serverVersion,
                'expected_step_version': step.serverVersion,
                'client_event_id': clientEventId,
                'device_id': deviceId,
                'created_at_device': now.toIso8601String(),
                'payload_version': 1,
                'base_server_version': execution.serverVersion,
              }),
              nextAttemptAt: now,
              createdAtDevice: now,
              baseServerVersion: Value(execution.serverVersion),
            ),
          );
    });
    return clientEventId;
  }

  Future<String> recordObservation({
    required String executionId,
    String? componentId,
    String? property,
    String? status,
    required String narrative,
    required String deviceId,
    DateTime? observedAt,
  }) async {
    final clientEventId = _uuid.v4();
    final createdAt = DateTime.now().toUtc();
    final payload = <String, Object?>{
      'client_event_id': clientEventId,
      'device_id': deviceId,
      'created_at_device': createdAt.toIso8601String(),
      'component_id': componentId,
      'property': property,
      'status': status,
      'narrative': narrative,
      'source': 'TECHNICIAN',
      'verification_status': 'UNVERIFIED',
      'observed_at': (observedAt ?? createdAt).toUtc().toIso8601String(),
    };
    await database.transaction(() async {
      await database
          .into(database.localObservations)
          .insert(
            LocalObservationsCompanion.insert(
              clientEventId: clientEventId,
              executionId: executionId,
              componentId: Value(componentId),
              property: Value(property),
              status: Value(status),
              narrative: narrative,
              observedAt: observedAt ?? createdAt,
              createdAtDevice: createdAt,
            ),
          );
      await database
          .into(database.outboxEntries)
          .insert(
            OutboxEntriesCompanion.insert(
              id: _uuid.v4(),
              clientEventId: clientEventId,
              idempotencyKey: _uuid.v4(),
              operation: 'RECORD_OBSERVATION',
              path: '/api/v1/executions/$executionId/observations',
              payloadJson: jsonEncode(payload),
              nextAttemptAt: createdAt,
              createdAtDevice: createdAt,
            ),
          );
    });
    return clientEventId;
  }

  Future<String> queueAttachment({
    required String siteId,
    required String entityKind,
    required String entityId,
    required String localPath,
    required String filename,
    required String mimeType,
    required int sizeBytes,
    required String checksumSha256,
  }) async {
    final clientEventId = _uuid.v4();
    final createdAt = DateTime.now().toUtc();
    final payload = <String, Object?>{
      'site_id': siteId,
      'entity_kind': entityKind,
      'entity_id': entityId,
      'client_event_id': clientEventId,
      'original_filename': filename,
      'declared_mime': mimeType,
      'size_bytes': sizeBytes,
      'checksum_sha256': checksumSha256,
    };
    await database.transaction(() async {
      await database
          .into(database.localAttachments)
          .insert(
            LocalAttachmentsCompanion.insert(
              clientEventId: clientEventId,
              entityKind: entityKind,
              entityId: entityId,
              siteId: siteId,
              localPath: localPath,
              filename: filename,
              mimeType: mimeType,
              sizeBytes: sizeBytes,
              checksumSha256: checksumSha256,
              createdAtDevice: createdAt,
            ),
          );
      await database
          .into(database.outboxEntries)
          .insert(
            OutboxEntriesCompanion.insert(
              id: _uuid.v4(),
              clientEventId: clientEventId,
              idempotencyKey: _uuid.v4(),
              operation: 'CREATE_ATTACHMENT_MANIFEST',
              path: '/api/v1/attachments',
              payloadJson: jsonEncode(payload),
              nextAttemptAt: createdAt,
              createdAtDevice: createdAt,
            ),
          );
    });
    return clientEventId;
  }

  Future<CopilotRecommendation> requestRecommendation({
    required String incidentId,
    required String executionId,
  }) async {
    final http = _http;
    final token = await _tokens?.accessToken();
    if (http == null || token == null) {
      throw StateError(
        'Copilot requires a live authenticated connection. Offline field records remain safe.',
      );
    }
    final response = await http.post<Map<String, dynamic>>(
      '/api/v1/incidents/$incidentId/recommendations',
      data: {
        'execution_id': executionId,
        'question': 'What is the next safe non-intrusive inspection step?',
      },
      options: Options(
        headers: {
          'Authorization': 'Bearer $token',
          'Idempotency-Key': _uuid.v4(),
        },
      ),
    );
    final body = response.data;
    if (body == null) throw StateError('Copilot returned an empty response');
    return CopilotRecommendation.fromJson(body);
  }
}

class CopilotRecommendation {
  const CopilotRecommendation({
    required this.status,
    required this.recommendation,
    required this.riskLevel,
    required this.confidence,
    required this.unknowns,
    required this.evidence,
    required this.provenance,
  });

  factory CopilotRecommendation.fromJson(Map<String, dynamic> value) {
    final output = value['output'] as Map<String, dynamic>;
    final evidenceValues = value['evidence'] as List<dynamic>? ?? const [];
    return CopilotRecommendation(
      status: output['status'] as String,
      recommendation: output['recommendation'] as String? ?? '',
      riskLevel: output['risk_level'] as String,
      confidence: (output['confidence'] as num).toDouble(),
      unknowns: (output['unknowns'] as List<dynamic>).cast<String>(),
      evidence: evidenceValues
          .cast<Map<String, dynamic>>()
          .map(
            (item) => CopilotEvidence(
              id: item['id'] as String,
              title: item['title'] as String,
              locator: item['locator'] as String,
              authority: item['authority'] as String,
            ),
          )
          .toList(growable: false),
      provenance:
          '${value['provider']}/${value['model']} · ${value['prompt_version']}',
    );
  }

  final String status;
  final String recommendation;
  final String riskLevel;
  final double confidence;
  final List<String> unknowns;
  final List<CopilotEvidence> evidence;
  final String provenance;
}

class CopilotEvidence {
  const CopilotEvidence({
    required this.id,
    required this.title,
    required this.locator,
    required this.authority,
  });

  final String id;
  final String title;
  final String locator;
  final String authority;
}

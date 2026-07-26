import 'dart:convert';
import 'dart:io';

import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:dio/dio.dart';
import 'package:drift/drift.dart';

import '../core/access_token.dart';
import '../data/database.dart';
import 'sync_policy.dart';

class SyncEngine {
  SyncEngine({
    required this.database,
    required this.http,
    required this.tokens,
    required this.connectivity,
    this.policy = const SyncPolicy(),
  });

  final MaintenanceDatabase database;
  final Dio http;
  final AccessTokenProvider tokens;
  final Connectivity connectivity;
  final SyncPolicy policy;
  bool _running = false;

  Future<void> runOnce() async {
    if (_running) return;
    final connectivityState = await connectivity.checkConnectivity();
    if (connectivityState.every((state) => state == ConnectivityResult.none)) {
      return;
    }
    _running = true;
    try {
      final now = DateTime.now().toUtc();
      final pending =
          await (database.select(database.outboxEntries)
                ..where(
                  (row) =>
                      row.status.isIn(const ['PENDING', 'RETRY']) &
                      row.nextAttemptAt.isSmallerOrEqualValue(now),
                )
                ..orderBy([
                  (row) => OrderingTerm.asc(row.createdAtDevice),
                  (row) => OrderingTerm.asc(row.id),
                ])
                ..limit(25))
              .get();
      for (final entry in pending) {
        await _send(entry);
      }
    } finally {
      _running = false;
    }
  }

  Future<void> _send(OutboxEntry entry) async {
    final token = await tokens.accessToken();
    if (token == null) return;
    try {
      if (entry.operation == 'CREATE_ATTACHMENT_MANIFEST') {
        await _sendAttachment(entry, token);
        return;
      }
      final response = await http.post<Map<String, dynamic>>(
        entry.path,
        data: jsonDecode(entry.payloadJson),
        options: Options(
          headers: {
            'Authorization': 'Bearer $token',
            'Idempotency-Key': entry.idempotencyKey,
          },
          validateStatus: (status) => status != null && status < 500,
        ),
      );
      final status = response.statusCode ?? 0;
      final disposition = policy.classifyStatus(status);
      if (disposition == SyncDisposition.applied) {
        await database.transaction(() async {
          await (database.update(
            database.outboxEntries,
          )..where((row) => row.id.equals(entry.id))).write(
            const OutboxEntriesCompanion(
              status: Value('APPLIED'),
              lastError: Value(null),
            ),
          );
          if (entry.operation == 'RECORD_MEASUREMENT') {
            await (database.update(database.localMeasurements)..where(
                  (row) => row.clientEventId.equals(entry.clientEventId),
                ))
                .write(
                  LocalMeasurementsCompanion(
                    syncStatus: const Value('SYNCED'),
                    serverId: Value(response.data?['id'] as String?),
                    syncError: const Value(null),
                  ),
                );
          } else if (entry.operation == 'RECORD_OBSERVATION') {
            await (database.update(database.localObservations)..where(
                  (row) => row.clientEventId.equals(entry.clientEventId),
                ))
                .write(
                  LocalObservationsCompanion(
                    syncStatus: const Value('SYNCED'),
                    serverId: Value(response.data?['id'] as String?),
                    syncError: const Value(null),
                  ),
                );
          }
        });
        return;
      }
      if (disposition == SyncDisposition.conflict) {
        await _mark(
          entry,
          'CONFLICT',
          'Server version or idempotency conflict',
        );
        return;
      }
      if (disposition == SyncDisposition.retry) {
        await _retry(entry, 'Server requested retry ($status)');
        return;
      }
      if (disposition == SyncDisposition.rejected) {
        await _mark(entry, 'REJECTED', 'Server rejected command ($status)');
      }
    } on DioException catch (error) {
      await _retry(entry, error.message ?? 'Network error');
    }
  }

  Future<void> _sendAttachment(OutboxEntry entry, String token) async {
    final local =
        await (database.select(database.localAttachments)
              ..where((row) => row.clientEventId.equals(entry.clientEventId)))
            .getSingle();
    final file = File(local.localPath);
    if (!await file.exists() || await file.length() != local.sizeBytes) {
      await _mark(entry, 'REJECTED', 'Local attachment is missing or changed');
      return;
    }
    final manifest = await http.post<Map<String, dynamic>>(
      entry.path,
      data: jsonDecode(entry.payloadJson),
      options: Options(
        headers: {
          'Authorization': 'Bearer $token',
          'Idempotency-Key': entry.idempotencyKey,
        },
        validateStatus: (status) => status != null && status < 500,
      ),
    );
    final manifestStatus = manifest.statusCode ?? 0;
    final manifestDisposition = policy.classifyStatus(manifestStatus);
    if (manifestDisposition == SyncDisposition.conflict) {
      await _mark(entry, 'CONFLICT', 'Attachment manifest conflict');
      return;
    }
    if (manifestDisposition == SyncDisposition.retry) {
      await _retry(entry, 'Attachment manifest retry ($manifestStatus)');
      return;
    }
    if (manifestDisposition == SyncDisposition.rejected) {
      await _mark(
        entry,
        'REJECTED',
        'Attachment manifest rejected ($manifestStatus)',
      );
      return;
    }
    if (manifest.data == null) {
      await _retry(entry, 'Attachment manifest response was empty');
      return;
    }
    final attachmentId = manifest.data!['id'] as String;
    final uploadUrl = manifest.data!['upload_url'] as String;
    final rawHeaders =
        manifest.data!['upload_headers'] as Map<String, dynamic>? ?? const {};
    final uploadHeaders = <String, Object?>{
      for (final item in rawHeaders.entries) item.key: item.value.toString(),
      Headers.contentLengthHeader: local.sizeBytes,
      Headers.contentTypeHeader: local.mimeType,
    };
    final upload = await Dio().put<void>(
      uploadUrl,
      data: file.openRead(),
      options: Options(
        headers: uploadHeaders,
        validateStatus: (status) => status != null && status < 500,
      ),
    );
    final uploadStatus = upload.statusCode ?? 0;
    final uploadDisposition = policy.classifyStatus(uploadStatus);
    if (uploadDisposition == SyncDisposition.retry) {
      await _retry(entry, 'Object upload retry ($uploadStatus)');
      return;
    }
    if (uploadDisposition != SyncDisposition.applied) {
      await _mark(entry, 'REJECTED', 'Object upload rejected ($uploadStatus)');
      return;
    }
    final completed = await http.post<Map<String, dynamic>>(
      '/api/v1/attachments/$attachmentId/complete',
      options: Options(
        headers: {
          'Authorization': 'Bearer $token',
          'Idempotency-Key': entry.idempotencyKey,
        },
        validateStatus: (status) => status != null && status < 500,
      ),
    );
    final completedStatus = completed.statusCode ?? 0;
    final completedDisposition = policy.classifyStatus(completedStatus);
    if (completedDisposition == SyncDisposition.conflict) {
      await _mark(entry, 'CONFLICT', 'Attachment completion conflict');
      return;
    }
    if (completedDisposition == SyncDisposition.retry) {
      await _retry(entry, 'Attachment completion retry ($completedStatus)');
      return;
    }
    if (completedDisposition == SyncDisposition.rejected) {
      await _mark(
        entry,
        'REJECTED',
        'Attachment verification rejected ($completedStatus)',
      );
      return;
    }
    await database.transaction(() async {
      await (database.update(
        database.outboxEntries,
      )..where((row) => row.id.equals(entry.id))).write(
        const OutboxEntriesCompanion(
          status: Value('APPLIED'),
          lastError: Value(null),
        ),
      );
      await (database.update(
        database.localAttachments,
      )..where((row) => row.clientEventId.equals(entry.clientEventId))).write(
        LocalAttachmentsCompanion(
          syncStatus: const Value('SYNCED'),
          serverId: Value(attachmentId),
          syncError: const Value(null),
        ),
      );
    });
  }

  Future<void> _retry(OutboxEntry entry, String reason) async {
    final attempts = entry.attemptCount + 1;
    final delay = policy.retryDelay(attempts);
    await (database.update(
      database.outboxEntries,
    )..where((row) => row.id.equals(entry.id))).write(
      OutboxEntriesCompanion(
        status: const Value('RETRY'),
        attemptCount: Value(attempts),
        lastAttemptAt: Value(DateTime.now().toUtc()),
        nextAttemptAt: Value(
          DateTime.now().toUtc().add(delay),
        ),
        lastError: Value(reason),
      ),
    );
  }

  Future<void> _mark(OutboxEntry entry, String status, String reason) async {
    await (database.update(
      database.outboxEntries,
    )..where((row) => row.id.equals(entry.id))).write(
      OutboxEntriesCompanion(
        status: Value(status),
        lastAttemptAt: Value(DateTime.now().toUtc()),
        lastError: Value(reason),
      ),
    );
  }
}

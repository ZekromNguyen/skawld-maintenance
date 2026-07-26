import 'dart:io';

import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:file_selector/file_selector.dart';
import 'package:flutter/material.dart';

import 'core/app_config.dart';
import 'core/auth_service.dart';
import 'core/device_identity.dart';
import 'data/database.dart';
import 'data/technician_repository.dart';
import 'sync/pull_service.dart';
import 'sync/sync_coordinator.dart';
import 'sync/sync_engine.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  late final AppConfig config;
  try {
    config = await AppConfig.load();
  } on Object catch (error) {
    runApp(ConfigurationFailure(error: error));
    return;
  }
  final database = MaintenanceDatabase();
  final deviceId = await DeviceIdentity().getOrCreate();
  final http = Dio(BaseOptions(baseUrl: config.apiUrl));
  final auth = AuthService(
    issuer: config.oidcIssuer,
    clientId: config.oidcClientId,
  );
  final connectivity = Connectivity();
  final coordinator = SyncCoordinator(
    connectivity: connectivity,
    pull: PullService(database: database, http: http, tokens: auth),
    push: SyncEngine(
      database: database,
      http: http,
      tokens: auth,
      connectivity: connectivity,
    ),
  )..start();
  runApp(
    SkawldMobile(
      repository: TechnicianRepository(database, http: http, tokens: auth),
      deviceId: deviceId,
      auth: auth,
      sync: coordinator,
    ),
  );
}

class ConfigurationFailure extends StatelessWidget {
  const ConfigurationFailure({required this.error, super.key});

  final Object error;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      home: Scaffold(
        body: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 560),
            child: Padding(
              padding: const EdgeInsets.all(32),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.warning_amber, size: 48),
                  const SizedBox(height: 16),
                  const Text(
                    'Desktop configuration is invalid',
                    style: TextStyle(fontSize: 20, fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    '$error\n\nCheck skawld-config.json or the managed environment variables.',
                    textAlign: TextAlign.center,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class SkawldMobile extends StatelessWidget {
  const SkawldMobile({
    required this.repository,
    required this.deviceId,
    required this.auth,
    required this.sync,
    super.key,
  });

  final TechnicianRepository repository;
  final String deviceId;
  final AuthService auth;
  final SyncCoordinator sync;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Skawld Maintenance',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF286B4D),
          brightness: Brightness.light,
        ),
        scaffoldBackgroundColor: const Color(0xFFF0F2F0),
        useMaterial3: true,
      ),
      home: SessionGate(
        repository: repository,
        deviceId: deviceId,
        auth: auth,
        sync: sync,
      ),
    );
  }
}

class SessionGate extends StatefulWidget {
  const SessionGate({
    required this.repository,
    required this.deviceId,
    required this.auth,
    required this.sync,
    super.key,
  });

  final TechnicianRepository repository;
  final String deviceId;
  final AuthService auth;
  final SyncCoordinator sync;

  @override
  State<SessionGate> createState() => _SessionGateState();
}

class _SessionGateState extends State<SessionGate> {
  bool? authenticated;
  bool busy = false;

  @override
  void initState() {
    super.initState();
    _restore();
  }

  Future<void> _restore() async {
    final token = await widget.auth.accessToken();
    final hasLocalSession =
        token != null || await widget.auth.hasLocalSession();
    if (!mounted) return;
    setState(() => authenticated = hasLocalSession);
    if (token != null) await widget.sync.synchronize();
  }

  Future<void> _login() async {
    setState(() => busy = true);
    final success = await widget.auth.authorize();
    if (!mounted) return;
    setState(() {
      busy = false;
      authenticated = success;
    });
    if (success) await widget.sync.synchronize();
  }

  @override
  Widget build(BuildContext context) {
    if (authenticated == null) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    if (authenticated == false) {
      return Scaffold(
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(32),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Icon(Icons.engineering, size: 48),
                const SizedBox(height: 16),
                const Text(
                  'Skawld Maintenance',
                  style: TextStyle(fontSize: 22, fontWeight: FontWeight.w700),
                ),
                const SizedBox(height: 8),
                const Text(
                  'Sign in while connected. Cached assigned work remains available during later outages.',
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 20),
                FilledButton(
                  onPressed: busy ? null : _login,
                  child: Text(busy ? 'Connecting…' : 'Sign in with OIDC'),
                ),
              ],
            ),
          ),
        ),
      );
    }
    return AssignedExecutionList(
      repository: widget.repository,
      deviceId: widget.deviceId,
      sync: widget.sync,
    );
  }
}

class AssignedExecutionList extends StatelessWidget {
  const AssignedExecutionList({
    required this.repository,
    required this.deviceId,
    required this.sync,
    super.key,
  });

  final TechnicianRepository repository;
  final String deviceId;
  final SyncCoordinator sync;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Assigned maintenance'),
            Text(
              'OFFLINE READY · ADVISORY ONLY',
              style: TextStyle(fontSize: 10, fontWeight: FontWeight.w700),
            ),
          ],
        ),
        actions: [
          IconButton(
            tooltip: 'Sync now',
            onPressed: () => sync.synchronize(),
            icon: const Icon(Icons.sync),
          ),
        ],
      ),
      body: StreamBuilder<List<CachedExecution>>(
        stream: repository.watchAssignedExecutions(),
        builder: (context, snapshot) {
          final executions = snapshot.data ?? const [];
          if (executions.isEmpty) {
            return const Center(
              child: Padding(
                padding: EdgeInsets.all(32),
                child: Text(
                  'No cached executions.\nConnect once to download your assigned work.',
                  textAlign: TextAlign.center,
                ),
              ),
            );
          }
          return ListView.separated(
            padding: const EdgeInsets.all(12),
            itemCount: executions.length,
            separatorBuilder: (_, _) => const SizedBox(height: 8),
            itemBuilder: (context, index) {
              final execution = executions[index];
              return Card(
                child: ListTile(
                  minVerticalPadding: 16,
                  leading: ExecutionStateMarker(state: execution.state),
                  title: Text(
                    '${execution.assetTag} · ${execution.purpose}',
                    style: const TextStyle(fontWeight: FontWeight.w700),
                  ),
                  subtitle: Text(execution.state.replaceAll('_', ' ')),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => Navigator.of(context).push(
                    MaterialPageRoute<void>(
                      builder: (_) => ExecutionDetail(
                        repository: repository,
                        execution: execution,
                        deviceId: deviceId,
                      ),
                    ),
                  ),
                ),
              );
            },
          );
        },
      ),
      bottomNavigationBar: const SafeArea(
        child: Padding(
          padding: EdgeInsets.all(12),
          child: Text(
            'Pending changes are stored on this device and sync automatically.',
            textAlign: TextAlign.center,
            style: TextStyle(fontSize: 11),
          ),
        ),
      ),
    );
  }
}

class ExecutionDetail extends StatefulWidget {
  const ExecutionDetail({
    required this.repository,
    required this.execution,
    required this.deviceId,
    super.key,
  });

  final TechnicianRepository repository;
  final CachedExecution execution;
  final String deviceId;

  @override
  State<ExecutionDetail> createState() => _ExecutionDetailState();
}

class _ExecutionDetailState extends State<ExecutionDetail> {
  final measurement = TextEditingController(text: '8.1');
  final note = TextEditingController();
  String measurementType = 'VIBRATION_VELOCITY';
  String? message;
  CopilotRecommendation? recommendation;
  bool copilotBusy = false;

  @override
  void dispose() {
    measurement.dispose();
    note.dispose();
    super.dispose();
  }

  Future<void> run(Future<void> Function() action) async {
    try {
      await action();
      if (mounted) setState(() => message = 'Saved locally · pending sync');
    } on Object catch (error) {
      if (mounted) setState(() => message = error.toString());
    }
  }

  Future<void> pickAttachment() async {
    const acceptedFiles = XTypeGroup(
      label: 'Photos and voice notes',
      extensions: ['jpg', 'jpeg', 'png', 'webp', 'm4a', 'mp4', 'mp3', 'wav'],
    );
    final selected = await openFile(acceptedTypeGroups: const [acceptedFiles]);
    if (selected == null) return;
    final file = File(selected.path);
    final size = await file.length();
    if (size <= 0 || size > 50 << 20) {
      throw StateError('Attachment must be between 1 byte and 50 MiB');
    }
    final digest = await sha256.bind(file.openRead()).first;
    await widget.repository.queueAttachment(
      siteId: widget.execution.siteId,
      entityKind: 'EXECUTION',
      entityId: widget.execution.id,
      localPath: file.path,
      filename: selected.name,
      mimeType: attachmentMIME(selected.name),
      sizeBytes: size,
      checksumSha256: digest.toString(),
    );
  }

  Future<void> requestRecommendation() async {
    setState(() {
      copilotBusy = true;
      message = null;
    });
    try {
      final value = await widget.repository.requestRecommendation(
        incidentId: widget.execution.incidentId,
        executionId: widget.execution.id,
      );
      if (mounted) setState(() => recommendation = value);
    } on Object catch (error) {
      if (mounted) setState(() => message = error.toString());
    } finally {
      if (mounted) setState(() => copilotBusy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(widget.execution.assetTag)),
      body: ListView(
        padding: const EdgeInsets.all(12),
        children: [
          Text(
            widget.execution.purpose,
            style: Theme.of(context).textTheme.titleLarge,
          ),
          const SizedBox(height: 4),
          const Text(
            'Skawld guides and records work. External PTW/LOTO remains authoritative.',
            style: TextStyle(fontSize: 11),
          ),
          if (widget.execution.state == 'ASSIGNED') ...[
            const SizedBox(height: 16),
            FilledButton(
              onPressed: () => run(
                () => widget.repository.startExecution(
                  widget.execution.id,
                  deviceId: widget.deviceId,
                ),
              ),
              child: const Text('Start inspection offline'),
            ),
          ],
          const SizedBox(height: 16),
          StreamBuilder<List<CachedExecutionStep>>(
            stream: widget.repository.watchSteps(widget.execution.id),
            builder: (context, snapshot) {
              final steps = snapshot.data ?? const [];
              return Column(
                children: steps
                    .map(
                      (step) => Card(
                        child: ListTile(
                          leading: CircleAvatar(
                            child: step.state == 'COMPLETED'
                                ? const Icon(Icons.check, size: 18)
                                : Text('${step.sequence}'),
                          ),
                          title: Text(step.title),
                          subtitle: Text(
                            [
                              step.riskLevel.replaceAll('_', ' '),
                              if (step.requiredPrerequisite != null)
                                'Requires ${step.requiredPrerequisite!.replaceAll('_', ' ')}',
                            ].join('\n'),
                          ),
                          trailing: TextButton(
                            onPressed: step.state == 'COMPLETED'
                                ? null
                                : () => run(
                                    () => widget.repository.completeStep(
                                      executionId: widget.execution.id,
                                      stepId: step.id,
                                      deviceId: widget.deviceId,
                                    ),
                                  ),
                            child: const Text('Complete'),
                          ),
                        ),
                      ),
                    )
                    .toList(),
              );
            },
          ),
          const SizedBox(height: 12),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Text(
                    'Record measurement',
                    style: TextStyle(fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 10),
                  DropdownButtonFormField<String>(
                    initialValue: measurementType,
                    items: const [
                      DropdownMenuItem(
                        value: 'VIBRATION_VELOCITY',
                        child: Text('Vibration velocity · mm/s'),
                      ),
                      DropdownMenuItem(
                        value: 'TEMPERATURE',
                        child: Text('Bearing temperature · °C'),
                      ),
                    ],
                    onChanged: (value) =>
                        setState(() => measurementType = value!),
                  ),
                  const SizedBox(height: 8),
                  TextField(
                    controller: measurement,
                    keyboardType: const TextInputType.numberWithOptions(
                      decimal: true,
                    ),
                    decoration: const InputDecoration(labelText: 'Exact value'),
                  ),
                  const SizedBox(height: 10),
                  FilledButton.tonal(
                    onPressed: () => run(() async {
                      await widget.repository.recordMeasurement(
                        executionId: widget.execution.id,
                        measurementType: measurementType,
                        value: measurement.text,
                        unit: measurementType == 'TEMPERATURE'
                            ? 'DEG_C'
                            : 'MM_PER_S',
                        deviceId: widget.deviceId,
                      );
                    }),
                    child: const Text('Save measurement locally'),
                  ),
                ],
              ),
            ),
          ),
          Card(
            color: const Color(0xFFF5F8F6),
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Text(
                    'Evidence-backed copilot',
                    style: TextStyle(fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 5),
                  const Text(
                    'Online only. Advisory proposals cannot complete workflow steps or authorize PTW/LOTO.',
                    style: TextStyle(fontSize: 11),
                  ),
                  const SizedBox(height: 10),
                  OutlinedButton.icon(
                    onPressed: copilotBusy ? null : requestRecommendation,
                    icon: const Icon(Icons.manage_search),
                    label: Text(
                      copilotBusy
                          ? 'Retrieving eligible evidence…'
                          : 'Recommend next inspection',
                    ),
                  ),
                  if (recommendation case final value?) ...[
                    const Divider(height: 24),
                    Row(
                      children: [
                        Chip(label: Text(value.riskLevel)),
                        const SizedBox(width: 8),
                        Text(
                          '${(value.confidence * 100).round()}% confidence',
                          style: const TextStyle(fontSize: 11),
                        ),
                      ],
                    ),
                    Text(
                      value.recommendation.isEmpty
                          ? 'INSUFFICIENT EVIDENCE'
                          : value.recommendation,
                      style: const TextStyle(fontWeight: FontWeight.w600),
                    ),
                    const SizedBox(height: 8),
                    ...value.evidence.map(
                      (item) => ListTile(
                        dense: true,
                        contentPadding: EdgeInsets.zero,
                        leading: const Icon(Icons.link, size: 18),
                        title: Text('${item.title} · ${item.locator}'),
                        subtitle: Text(item.authority),
                      ),
                    ),
                    Text(
                      'Unknowns: ${value.unknowns.join(' · ')}',
                      style: const TextStyle(fontSize: 10),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      '${value.provenance} · human confirmation required',
                      style: const TextStyle(
                        fontSize: 9,
                        fontFamily: 'monospace',
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Text(
                    'Technician observation',
                    style: TextStyle(fontWeight: FontWeight.w700),
                  ),
                  TextField(
                    controller: note,
                    minLines: 2,
                    maxLines: 5,
                    decoration: const InputDecoration(
                      hintText: 'What did you observe?',
                    ),
                  ),
                  const SizedBox(height: 10),
                  FilledButton.tonal(
                    onPressed: () => run(() async {
                      await widget.repository.recordObservation(
                        executionId: widget.execution.id,
                        narrative: note.text,
                        deviceId: widget.deviceId,
                      );
                      note.clear();
                    }),
                    child: const Text('Save note locally'),
                  ),
                ],
              ),
            ),
          ),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Text(
                    'Photo or voice evidence',
                    style: TextStyle(fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 6),
                  const Text(
                    'The file is queued separately. A failed upload does not remove the inspection record.',
                    style: TextStyle(fontSize: 11),
                  ),
                  const SizedBox(height: 10),
                  OutlinedButton.icon(
                    onPressed: () => run(pickAttachment),
                    icon: const Icon(Icons.attach_file),
                    label: const Text('Choose file from this device'),
                  ),
                ],
              ),
            ),
          ),
          if (message != null)
            Padding(
              padding: const EdgeInsets.all(8),
              child: Text(message!, textAlign: TextAlign.center),
            ),
        ],
      ),
    );
  }
}

String attachmentMIME(String filename) {
  final extension = filename.toLowerCase().split('.').last;
  return switch (extension) {
    'jpg' || 'jpeg' => 'image/jpeg',
    'png' => 'image/png',
    'webp' => 'image/webp',
    'm4a' => 'audio/m4a',
    'mp4' => 'audio/mp4',
    'mp3' => 'audio/mpeg',
    'wav' => 'audio/wav',
    _ => throw StateError('Unsupported attachment type'),
  };
}

class ExecutionStateMarker extends StatelessWidget {
  const ExecutionStateMarker({required this.state, super.key});

  final String state;

  @override
  Widget build(BuildContext context) {
    final color = switch (state) {
      'IN_PROGRESS' => const Color(0xFFC36935),
      'COMPLETED' => const Color(0xFF3B855F),
      _ => const Color(0xFF6D7C74),
    };
    return Container(width: 6, height: 48, color: color);
  }
}

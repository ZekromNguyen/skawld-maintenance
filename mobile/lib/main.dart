import 'dart:io';

import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:file_selector/file_selector.dart';
import 'package:flutter/material.dart';

import 'core/app_config.dart';
import 'core/auth_service.dart';
import 'core/device_identity.dart';
import 'core/locale_controller.dart';
import 'core/repository_errors.dart';
import 'data/database.dart';
import 'data/technician_repository.dart';
import 'l10n/app_localizations.dart';
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
  final localeController = await LocaleController.load();
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
      localeController: localeController,
    ),
  );
}

class ConfigurationFailure extends StatelessWidget {
  const ConfigurationFailure({required this.error, super.key});

  final Object error;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      supportedLocales: AppLocalizations.supportedLocales,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
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
                  Text(
                    l10n.config_invalid,
                    style: const TextStyle(
                      fontSize: 20,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    '$error\n\n${l10n.config_hint}',
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
    required this.localeController,
    super.key,
  });

  final TechnicianRepository repository;
  final String deviceId;
  final AuthService auth;
  final SyncCoordinator sync;
  final LocaleController localeController;

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<Locale?>(
      valueListenable: localeController.locale,
      builder: (context, locale, _) => MaterialApp(
        onGenerateTitle: (context) => AppLocalizations.of(context).appTitle,
        debugShowCheckedModeBanner: false,
        locale: locale,
        supportedLocales: AppLocalizations.supportedLocales,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
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
          localeController: localeController,
        ),
      ),
    );
  }
}

class LanguageMenu extends StatelessWidget {
  const LanguageMenu({required this.controller, super.key});

  final LocaleController controller;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return ValueListenableBuilder<Locale?>(
      valueListenable: controller.locale,
      builder: (context, locale, _) => PopupMenuButton<String>(
        icon: const Icon(Icons.translate),
        tooltip: l10n.language,
        initialValue: locale?.languageCode ?? '',
        onSelected: controller.setLanguage,
        itemBuilder: (_) => [
          PopupMenuItem(value: '', child: Text(l10n.locale_system)),
          const PopupMenuItem(value: 'en', child: Text('English')),
          const PopupMenuItem(value: 'vi', child: Text('Tiếng Việt')),
        ],
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
    required this.localeController,
    super.key,
  });

  final TechnicianRepository repository;
  final String deviceId;
  final AuthService auth;
  final SyncCoordinator sync;
  final LocaleController localeController;

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

  Future<void> _signOut() async {
    await widget.auth.signOutLocal();
    if (!mounted) return;
    setState(() => authenticated = false);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    if (authenticated == null) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    if (authenticated == false) {
      return Scaffold(
        appBar: AppBar(
          actions: [LanguageMenu(controller: widget.localeController)],
        ),
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
                Text(l10n.login_subtitle, textAlign: TextAlign.center),
                const SizedBox(height: 20),
                FilledButton(
                  onPressed: busy ? null : _login,
                  child: Text(busy ? l10n.login_connecting : l10n.login_button),
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
      localeController: widget.localeController,
      onSignOut: _signOut,
    );
  }
}

class AssignedExecutionList extends StatelessWidget {
  const AssignedExecutionList({
    required this.repository,
    required this.deviceId,
    required this.sync,
    required this.localeController,
    required this.onSignOut,
    super.key,
  });

  final TechnicianRepository repository;
  final String deviceId;
  final SyncCoordinator sync;
  final LocaleController localeController;
  final VoidCallback onSignOut;

  Future<void> _confirmSignOut(BuildContext context) async {
    final l10n = AppLocalizations.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(l10n.signout_title),
        content: Text(l10n.signout_message),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: Text(l10n.signout_cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: Text(l10n.signout_confirm),
          ),
        ],
      ),
    );
    if (confirmed == true) onSignOut();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l10n.list_assignedMaintenance),
            Text(
              l10n.list_offlineBanner,
              style: const TextStyle(fontSize: 10, fontWeight: FontWeight.w700),
            ),
          ],
        ),
        actions: [
          LanguageMenu(controller: localeController),
          IconButton(
            tooltip: l10n.list_syncNow,
            onPressed: () => sync.synchronize(),
            icon: const Icon(Icons.sync),
          ),
          IconButton(
            tooltip: l10n.signout_tooltip,
            onPressed: () => _confirmSignOut(context),
            icon: const Icon(Icons.logout),
          ),
        ],
      ),
      body: StreamBuilder<List<CachedExecution>>(
        stream: repository.watchAssignedExecutions(),
        builder: (context, snapshot) {
          final executions = snapshot.data ?? const [];
          if (executions.isEmpty) {
            return Center(
              child: Padding(
                padding: const EdgeInsets.all(32),
                child: Text(l10n.list_empty, textAlign: TextAlign.center),
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
      bottomNavigationBar: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Text(
            l10n.list_footer,
            textAlign: TextAlign.center,
            style: const TextStyle(fontSize: 11),
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

  String _localizeError(Object error, AppLocalizations l10n) {
    if (error is RepositoryException) {
      return switch (error.code) {
        RepositoryErrorCode.notAssigned => l10n.error_notAssigned,
        RepositoryErrorCode.notInProgress => l10n.error_notInProgress,
        RepositoryErrorCode.prerequisiteUnverified =>
          l10n.error_prerequisiteUnverified(
            error.params!['prerequisite']! as String,
          ),
        RepositoryErrorCode.copilotOffline => l10n.error_copilotOffline,
        RepositoryErrorCode.copilotEmptyResponse => l10n.error_copilotEmpty,
        RepositoryErrorCode.attachmentSizeOutOfRange =>
          l10n.error_attachmentSize,
        RepositoryErrorCode.unsupportedAttachmentType =>
          l10n.error_attachmentType,
      };
    }
    return error.toString();
  }

  Future<void> run(Future<void> Function() action) async {
    final l10n = AppLocalizations.of(context);
    try {
      await action();
      if (mounted) setState(() => message = l10n.savedPendingSync);
    } on Object catch (error) {
      if (mounted) setState(() => message = _localizeError(error, l10n));
    }
  }

  Future<void> pickAttachment() async {
    final l10n = AppLocalizations.of(context);
    final acceptedFiles = XTypeGroup(
      label: l10n.attachment_pickerLabel,
      extensions: const [
        'jpg',
        'jpeg',
        'png',
        'webp',
        'm4a',
        'mp4',
        'mp3',
        'wav',
      ],
    );
    final selected = await openFile(acceptedTypeGroups: [acceptedFiles]);
    if (selected == null) return;
    final file = File(selected.path);
    final size = await file.length();
    if (size <= 0 || size > 50 << 20) {
      throw RepositoryException(
        RepositoryErrorCode.attachmentSizeOutOfRange,
        'Attachment must be between 1 byte and 50 MiB',
      );
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
    final l10n = AppLocalizations.of(context);
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
      if (mounted) setState(() => message = _localizeError(error, l10n));
    } finally {
      if (mounted) setState(() => copilotBusy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
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
          Text(l10n.detail_disclaimer, style: const TextStyle(fontSize: 11)),
          if (widget.execution.state == 'ASSIGNED') ...[
            const SizedBox(height: 16),
            FilledButton(
              onPressed: () => run(
                () => widget.repository.startExecution(
                  widget.execution.id,
                  deviceId: widget.deviceId,
                ),
              ),
              child: Text(l10n.detail_startOffline),
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
                                l10n.detail_requiresPrerequisite(
                                  step.requiredPrerequisite!.replaceAll(
                                    '_',
                                    ' ',
                                  ),
                                ),
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
                            child: Text(l10n.detail_complete),
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
                  Text(
                    l10n.measurement_record,
                    style: const TextStyle(fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 10),
                  DropdownButtonFormField<String>(
                    initialValue: measurementType,
                    items: [
                      DropdownMenuItem(
                        value: 'VIBRATION_VELOCITY',
                        child: Text(l10n.measurement_vibrationVelocity),
                      ),
                      DropdownMenuItem(
                        value: 'TEMPERATURE',
                        child: Text(l10n.measurement_bearingTemperature),
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
                    decoration: InputDecoration(
                      labelText: l10n.measurement_exactValue,
                    ),
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
                    child: Text(l10n.measurement_save),
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
                  Text(
                    l10n.copilot_title,
                    style: const TextStyle(fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 5),
                  Text(
                    l10n.copilot_disclaimer,
                    style: const TextStyle(fontSize: 11),
                  ),
                  const SizedBox(height: 10),
                  OutlinedButton.icon(
                    onPressed: copilotBusy ? null : requestRecommendation,
                    icon: const Icon(Icons.manage_search),
                    label: Text(
                      copilotBusy ? l10n.copilot_busy : l10n.copilot_recommend,
                    ),
                  ),
                  if (recommendation case final value?) ...[
                    const Divider(height: 24),
                    Row(
                      children: [
                        Chip(label: Text(value.riskLevel)),
                        const SizedBox(width: 8),
                        Text(
                          l10n.copilot_confidence(
                            (value.confidence * 100).round(),
                          ),
                          style: const TextStyle(fontSize: 11),
                        ),
                      ],
                    ),
                    Text(
                      value.recommendation.isEmpty
                          ? l10n.copilot_insufficient
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
                      l10n.copilot_unknowns(value.unknowns.join(' · ')),
                      style: const TextStyle(fontSize: 10),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      l10n.copilot_humanConfirmation(value.provenance),
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
                  Text(
                    l10n.observation_title,
                    style: const TextStyle(fontWeight: FontWeight.w700),
                  ),
                  TextField(
                    controller: note,
                    minLines: 2,
                    maxLines: 5,
                    decoration: InputDecoration(
                      hintText: l10n.observation_hint,
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
                    child: Text(l10n.observation_save),
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
                  Text(
                    l10n.attachment_title,
                    style: const TextStyle(fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 6),
                  Text(
                    l10n.attachment_subtitle,
                    style: const TextStyle(fontSize: 11),
                  ),
                  const SizedBox(height: 10),
                  OutlinedButton.icon(
                    onPressed: () => run(pickAttachment),
                    icon: const Icon(Icons.attach_file),
                    label: Text(l10n.attachment_choose),
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
    _ => throw RepositoryException(
      RepositoryErrorCode.unsupportedAttachmentType,
      'Unsupported attachment type',
    ),
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

// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'Skawld Maintenance';

  @override
  String get language => 'Language';

  @override
  String get locale_system => 'System';

  @override
  String get locale_english => 'English';

  @override
  String get locale_vietnamese => 'Tiếng Việt';

  @override
  String get login_subtitle =>
      'Sign in while connected. Cached assigned work remains available during later outages.';

  @override
  String get login_button => 'Sign in with OIDC';

  @override
  String get login_connecting => 'Connecting…';

  @override
  String get list_assignedMaintenance => 'Assigned maintenance';

  @override
  String get list_offlineBanner => 'OFFLINE READY · ADVISORY ONLY';

  @override
  String get list_syncNow => 'Sync now';

  @override
  String get list_empty =>
      'No cached executions.\nConnect once to download your assigned work.';

  @override
  String get list_footer =>
      'Pending changes are stored on this device and sync automatically.';

  @override
  String get signout_title => 'Sign out?';

  @override
  String get signout_message =>
      'Your session on this device will be ended. Cached work stays on this device and syncs after you sign back in.';

  @override
  String get signout_cancel => 'Cancel';

  @override
  String get signout_confirm => 'Sign out';

  @override
  String get signout_tooltip => 'Sign out';

  @override
  String get detail_disclaimer =>
      'Skawld guides and records work. External PTW/LOTO remains authoritative.';

  @override
  String get detail_startOffline => 'Start inspection offline';

  @override
  String detail_requiresPrerequisite(String prerequisite) {
    return 'Requires $prerequisite';
  }

  @override
  String get detail_complete => 'Complete';

  @override
  String get measurement_record => 'Record measurement';

  @override
  String get measurement_vibrationVelocity => 'Vibration velocity · mm/s';

  @override
  String get measurement_bearingTemperature => 'Bearing temperature · °C';

  @override
  String get measurement_exactValue => 'Exact value';

  @override
  String get measurement_save => 'Save measurement locally';

  @override
  String get copilot_title => 'Evidence-backed copilot';

  @override
  String get copilot_disclaimer =>
      'Online only. Advisory proposals cannot complete workflow steps or authorize PTW/LOTO.';

  @override
  String get copilot_busy => 'Retrieving eligible evidence…';

  @override
  String get copilot_recommend => 'Recommend next inspection';

  @override
  String copilot_confidence(int percent) {
    return '$percent% confidence';
  }

  @override
  String get copilot_insufficient => 'INSUFFICIENT EVIDENCE';

  @override
  String copilot_unknowns(String value) {
    return 'Unknowns: $value';
  }

  @override
  String copilot_humanConfirmation(String provenance) {
    return '$provenance · human confirmation required';
  }

  @override
  String get observation_title => 'Technician observation';

  @override
  String get observation_hint => 'What did you observe?';

  @override
  String get observation_save => 'Save note locally';

  @override
  String get attachment_title => 'Photo or voice evidence';

  @override
  String get attachment_subtitle =>
      'The file is queued separately. A failed upload does not remove the inspection record.';

  @override
  String get attachment_choose => 'Choose file from this device';

  @override
  String get attachment_pickerLabel => 'Photos and voice notes';

  @override
  String get savedPendingSync => 'Saved locally · pending sync';

  @override
  String get error_notAssigned => 'Only an assigned execution can start';

  @override
  String get error_notInProgress => 'Execution is not in progress';

  @override
  String error_prerequisiteUnverified(String prerequisite) {
    return '$prerequisite must be verified by the external authority while online';
  }

  @override
  String get error_copilotOffline =>
      'Copilot requires a live authenticated connection. Offline field records remain safe.';

  @override
  String get error_copilotEmpty => 'Copilot returned an empty response';

  @override
  String get error_attachmentSize =>
      'Attachment must be between 1 byte and 50 MiB';

  @override
  String get error_attachmentType => 'Unsupported attachment type';

  @override
  String get config_invalid => 'Desktop configuration is invalid';

  @override
  String get config_hint =>
      'Check skawld-config.json or the managed environment variables.';
}

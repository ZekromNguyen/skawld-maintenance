import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_vi.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('vi'),
  ];

  /// No description provided for @appTitle.
  ///
  /// In en, this message translates to:
  /// **'Skawld Maintenance'**
  String get appTitle;

  /// No description provided for @language.
  ///
  /// In en, this message translates to:
  /// **'Language'**
  String get language;

  /// No description provided for @locale_system.
  ///
  /// In en, this message translates to:
  /// **'System'**
  String get locale_system;

  /// No description provided for @locale_english.
  ///
  /// In en, this message translates to:
  /// **'English'**
  String get locale_english;

  /// No description provided for @locale_vietnamese.
  ///
  /// In en, this message translates to:
  /// **'Tiếng Việt'**
  String get locale_vietnamese;

  /// No description provided for @login_subtitle.
  ///
  /// In en, this message translates to:
  /// **'Sign in while connected. Cached assigned work remains available during later outages.'**
  String get login_subtitle;

  /// No description provided for @login_button.
  ///
  /// In en, this message translates to:
  /// **'Sign in with OIDC'**
  String get login_button;

  /// No description provided for @login_connecting.
  ///
  /// In en, this message translates to:
  /// **'Connecting…'**
  String get login_connecting;

  /// No description provided for @list_assignedMaintenance.
  ///
  /// In en, this message translates to:
  /// **'Assigned maintenance'**
  String get list_assignedMaintenance;

  /// No description provided for @list_offlineBanner.
  ///
  /// In en, this message translates to:
  /// **'OFFLINE READY · ADVISORY ONLY'**
  String get list_offlineBanner;

  /// No description provided for @list_syncNow.
  ///
  /// In en, this message translates to:
  /// **'Sync now'**
  String get list_syncNow;

  /// No description provided for @list_empty.
  ///
  /// In en, this message translates to:
  /// **'No cached executions.\nConnect once to download your assigned work.'**
  String get list_empty;

  /// No description provided for @list_footer.
  ///
  /// In en, this message translates to:
  /// **'Pending changes are stored on this device and sync automatically.'**
  String get list_footer;

  /// No description provided for @signout_title.
  ///
  /// In en, this message translates to:
  /// **'Sign out?'**
  String get signout_title;

  /// No description provided for @signout_message.
  ///
  /// In en, this message translates to:
  /// **'Your session on this device will be ended. Cached work stays on this device and syncs after you sign back in.'**
  String get signout_message;

  /// No description provided for @signout_cancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get signout_cancel;

  /// No description provided for @signout_confirm.
  ///
  /// In en, this message translates to:
  /// **'Sign out'**
  String get signout_confirm;

  /// No description provided for @signout_tooltip.
  ///
  /// In en, this message translates to:
  /// **'Sign out'**
  String get signout_tooltip;

  /// No description provided for @detail_disclaimer.
  ///
  /// In en, this message translates to:
  /// **'Skawld guides and records work. External PTW/LOTO remains authoritative.'**
  String get detail_disclaimer;

  /// No description provided for @detail_startOffline.
  ///
  /// In en, this message translates to:
  /// **'Start inspection offline'**
  String get detail_startOffline;

  /// No description provided for @detail_requiresPrerequisite.
  ///
  /// In en, this message translates to:
  /// **'Requires {prerequisite}'**
  String detail_requiresPrerequisite(String prerequisite);

  /// No description provided for @detail_complete.
  ///
  /// In en, this message translates to:
  /// **'Complete'**
  String get detail_complete;

  /// No description provided for @measurement_record.
  ///
  /// In en, this message translates to:
  /// **'Record measurement'**
  String get measurement_record;

  /// No description provided for @measurement_vibrationVelocity.
  ///
  /// In en, this message translates to:
  /// **'Vibration velocity · mm/s'**
  String get measurement_vibrationVelocity;

  /// No description provided for @measurement_bearingTemperature.
  ///
  /// In en, this message translates to:
  /// **'Bearing temperature · °C'**
  String get measurement_bearingTemperature;

  /// No description provided for @measurement_exactValue.
  ///
  /// In en, this message translates to:
  /// **'Exact value'**
  String get measurement_exactValue;

  /// No description provided for @measurement_save.
  ///
  /// In en, this message translates to:
  /// **'Save measurement locally'**
  String get measurement_save;

  /// No description provided for @copilot_title.
  ///
  /// In en, this message translates to:
  /// **'Evidence-backed copilot'**
  String get copilot_title;

  /// No description provided for @copilot_disclaimer.
  ///
  /// In en, this message translates to:
  /// **'Online only. Advisory proposals cannot complete workflow steps or authorize PTW/LOTO.'**
  String get copilot_disclaimer;

  /// No description provided for @copilot_busy.
  ///
  /// In en, this message translates to:
  /// **'Retrieving eligible evidence…'**
  String get copilot_busy;

  /// No description provided for @copilot_recommend.
  ///
  /// In en, this message translates to:
  /// **'Recommend next inspection'**
  String get copilot_recommend;

  /// No description provided for @copilot_confidence.
  ///
  /// In en, this message translates to:
  /// **'{percent}% confidence'**
  String copilot_confidence(int percent);

  /// No description provided for @copilot_insufficient.
  ///
  /// In en, this message translates to:
  /// **'INSUFFICIENT EVIDENCE'**
  String get copilot_insufficient;

  /// No description provided for @copilot_unknowns.
  ///
  /// In en, this message translates to:
  /// **'Unknowns: {value}'**
  String copilot_unknowns(String value);

  /// No description provided for @copilot_humanConfirmation.
  ///
  /// In en, this message translates to:
  /// **'{provenance} · human confirmation required'**
  String copilot_humanConfirmation(String provenance);

  /// No description provided for @observation_title.
  ///
  /// In en, this message translates to:
  /// **'Technician observation'**
  String get observation_title;

  /// No description provided for @observation_hint.
  ///
  /// In en, this message translates to:
  /// **'What did you observe?'**
  String get observation_hint;

  /// No description provided for @observation_save.
  ///
  /// In en, this message translates to:
  /// **'Save note locally'**
  String get observation_save;

  /// No description provided for @attachment_title.
  ///
  /// In en, this message translates to:
  /// **'Photo or voice evidence'**
  String get attachment_title;

  /// No description provided for @attachment_subtitle.
  ///
  /// In en, this message translates to:
  /// **'The file is queued separately. A failed upload does not remove the inspection record.'**
  String get attachment_subtitle;

  /// No description provided for @attachment_choose.
  ///
  /// In en, this message translates to:
  /// **'Choose file from this device'**
  String get attachment_choose;

  /// No description provided for @attachment_pickerLabel.
  ///
  /// In en, this message translates to:
  /// **'Photos and voice notes'**
  String get attachment_pickerLabel;

  /// No description provided for @savedPendingSync.
  ///
  /// In en, this message translates to:
  /// **'Saved locally · pending sync'**
  String get savedPendingSync;

  /// No description provided for @error_notAssigned.
  ///
  /// In en, this message translates to:
  /// **'Only an assigned execution can start'**
  String get error_notAssigned;

  /// No description provided for @error_notInProgress.
  ///
  /// In en, this message translates to:
  /// **'Execution is not in progress'**
  String get error_notInProgress;

  /// No description provided for @error_prerequisiteUnverified.
  ///
  /// In en, this message translates to:
  /// **'{prerequisite} must be verified by the external authority while online'**
  String error_prerequisiteUnverified(String prerequisite);

  /// No description provided for @error_copilotOffline.
  ///
  /// In en, this message translates to:
  /// **'Copilot requires a live authenticated connection. Offline field records remain safe.'**
  String get error_copilotOffline;

  /// No description provided for @error_copilotEmpty.
  ///
  /// In en, this message translates to:
  /// **'Copilot returned an empty response'**
  String get error_copilotEmpty;

  /// No description provided for @error_attachmentSize.
  ///
  /// In en, this message translates to:
  /// **'Attachment must be between 1 byte and 50 MiB'**
  String get error_attachmentSize;

  /// No description provided for @error_attachmentType.
  ///
  /// In en, this message translates to:
  /// **'Unsupported attachment type'**
  String get error_attachmentType;

  /// No description provided for @config_invalid.
  ///
  /// In en, this message translates to:
  /// **'Desktop configuration is invalid'**
  String get config_invalid;

  /// No description provided for @config_hint.
  ///
  /// In en, this message translates to:
  /// **'Check skawld-config.json or the managed environment variables.'**
  String get config_hint;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'vi'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
    case 'vi':
      return AppLocalizationsVi();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}

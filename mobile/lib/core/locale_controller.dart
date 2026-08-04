import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Resolves and persists the app language. A null locale means "follow the
/// system locale", which MaterialApp resolves against the supported locales
/// via [WidgetsBinding.instance.platformDispatcher.locale].
class LocaleController {
  LocaleController(this._prefs) {
    _restore();
  }

  static const _storageKey = 'skawld.locale';

  final SharedPreferences _prefs;
  final ValueNotifier<Locale?> locale = ValueNotifier<Locale?>(null);

  static Future<LocaleController> load() async {
    final prefs = await SharedPreferences.getInstance();
    return LocaleController(prefs);
  }

  void _restore() {
    final code = _prefs.getString(_storageKey);
    if (code != null && code.isNotEmpty) {
      locale.value = Locale(code);
    }
  }

  void setLanguage(String? code) {
    final effective = (code == null || code.isEmpty) ? '' : code;
    unawaited(_prefs.setString(_storageKey, effective));
    locale.value = effective.isEmpty ? null : Locale(effective);
  }

  void dispose() {
    locale.dispose();
  }
}

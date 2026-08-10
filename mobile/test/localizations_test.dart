import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:skawld_maintenance_mobile/l10n/app_localizations.dart';

void main() {
  testWidgets('Vietnamese localization resolves the login button', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('vi'),
        supportedLocales: AppLocalizations.supportedLocales,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        home: Builder(
          builder: (context) {
            final l10n = AppLocalizations.of(context);
            return Text(l10n.login_button);
          },
        ),
      ),
    );
    expect(find.text('Đăng nhập bằng OIDC'), findsOneWidget);
  });

  testWidgets('English localization is the default fallback', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        supportedLocales: AppLocalizations.supportedLocales,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        home: Builder(
          builder: (context) {
            final l10n = AppLocalizations.of(context);
            return Text(l10n.detail_startOffline);
          },
        ),
      ),
    );
    expect(find.text('Start inspection offline'), findsOneWidget);
  });
}

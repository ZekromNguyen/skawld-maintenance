import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:openid_client/openid_client_io.dart';
import 'package:url_launcher/url_launcher.dart';

import 'access_token.dart';

class AuthService implements AccessTokenProvider {
  AuthService({
    required this.issuer,
    required this.clientId,
    this.loopbackPort = 43821,
    FlutterSecureStorage? storage,
  }) : _storage = storage ?? const FlutterSecureStorage();

  final String issuer;
  final String clientId;
  final int loopbackPort;
  final FlutterSecureStorage _storage;

  static const _credentialKey = 'skawld.oidc.credential';
  Credential? _credential;

  Future<bool> authorize() async {
    final discovered = await Issuer.discover(Uri.parse(issuer));
    final client = Client(discovered, clientId);
    final authenticator = Authenticator(
      client,
      port: loopbackPort,
      scopes: const ['openid', 'profile', 'email', 'offline_access'],
      urlLancher: (url) async {
        final opened = await launchUrl(
          Uri.parse(url),
          mode: LaunchMode.externalApplication,
        );
        if (!opened) throw StateError('Could not open system browser');
      },
    );
    _credential = await authenticator.authorize();
    await _persist();
    return true;
  }

  Future<bool> hasLocalSession() async {
    return _credential != null ||
        (await _storage.read(key: _credentialKey)) != null;
  }

  @override
  Future<String?> accessToken() async {
    final credential = await _load();
    if (credential == null) return null;
    try {
      final token = await credential.getTokenResponse();
      await _persist();
      return token.accessToken;
    } on Object {
      return null;
    }
  }

  Future<void> signOutLocal() async {
    _credential = null;
    await _storage.delete(key: _credentialKey);
  }

  Future<Credential?> _load() async {
    if (_credential != null) return _credential;
    final encoded = await _storage.read(key: _credentialKey);
    if (encoded == null) return null;
    _credential = Credential.fromJson(
      jsonDecode(encoded) as Map<String, dynamic>,
    );
    return _credential;
  }

  Future<void> _persist() async {
    final credential = _credential;
    if (credential == null) return;
    await _storage.write(
      key: _credentialKey,
      value: jsonEncode(credential.toJson()),
    );
  }
}

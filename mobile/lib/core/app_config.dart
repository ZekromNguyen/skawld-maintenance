import 'dart:convert';
import 'dart:io';

class AppConfig {
  const AppConfig({
    required this.apiUrl,
    required this.oidcIssuer,
    required this.oidcClientId,
  });

  final String apiUrl;
  final String oidcIssuer;
  final String oidcClientId;

  static const _compiledApiUrl = String.fromEnvironment(
    'SKAWLD_API_URL',
    defaultValue: 'http://localhost:8080',
  );
  static const _compiledOIDCIssuer = String.fromEnvironment(
    'SKAWLD_OIDC_ISSUER',
    defaultValue: 'http://localhost:8081/realms/skawld',
  );
  static const _compiledOIDCClientID = String.fromEnvironment(
    'SKAWLD_OIDC_CLIENT_ID',
    defaultValue: 'skawld-mobile',
  );

  static Future<AppConfig> load() async {
    final explicitPath = Platform.environment['SKAWLD_CONFIG_FILE'];
    if (explicitPath != null &&
        explicitPath.trim().isNotEmpty &&
        !await File(explicitPath).exists()) {
      throw FileSystemException(
        'SKAWLD_CONFIG_FILE does not exist',
        explicitPath,
      );
    }
    Map<String, dynamic> fileValues = const {};
    for (final candidate in _candidateFiles()) {
      if (!await candidate.exists()) continue;
      final decoded = jsonDecode(await candidate.readAsString());
      if (decoded is! Map<String, dynamic>) {
        throw const FormatException(
          'Desktop configuration must be a JSON object',
        );
      }
      fileValues = decoded;
      break;
    }

    return fromValues(
      apiUrl:
          Platform.environment['SKAWLD_API_URL'] ??
          (fileValues['api_url'] as String?) ??
          _compiledApiUrl,
      oidcIssuer:
          Platform.environment['SKAWLD_OIDC_ISSUER'] ??
          (fileValues['oidc_issuer'] as String?) ??
          _compiledOIDCIssuer,
      oidcClientId:
          Platform.environment['SKAWLD_OIDC_CLIENT_ID'] ??
          (fileValues['oidc_client_id'] as String?) ??
          _compiledOIDCClientID,
    );
  }

  static AppConfig fromValues({
    required String apiUrl,
    required String oidcIssuer,
    required String oidcClientId,
  }) {
    final api = _httpURI(apiUrl, 'api_url');
    final issuer = _httpURI(oidcIssuer, 'oidc_issuer');
    final clientID = oidcClientId.trim();
    if (clientID.isEmpty) {
      throw const FormatException('oidc_client_id is required');
    }
    return AppConfig(
      apiUrl: _withoutTrailingSlash(api.toString()),
      oidcIssuer: _withoutTrailingSlash(issuer.toString()),
      oidcClientId: clientID,
    );
  }

  static Iterable<File> _candidateFiles() sync* {
    final explicit = Platform.environment['SKAWLD_CONFIG_FILE'];
    if (explicit != null && explicit.trim().isNotEmpty) {
      yield File(explicit);
    }
    yield File(
      '${Directory.current.path}${Platform.pathSeparator}skawld-config.json',
    );

    var installationDirectory = File(Platform.resolvedExecutable).parent;
    if (Platform.isMacOS &&
        installationDirectory.path.endsWith('${Platform.pathSeparator}MacOS')) {
      installationDirectory = installationDirectory.parent.parent.parent;
    }
    yield File(
      '${installationDirectory.path}${Platform.pathSeparator}skawld-config.json',
    );
  }

  static Uri _httpURI(String value, String field) {
    final uri = Uri.tryParse(value.trim());
    if (uri == null ||
        !uri.hasAuthority ||
        (uri.scheme != 'http' && uri.scheme != 'https')) {
      throw FormatException('$field must be an absolute HTTP(S) URL');
    }
    return uri;
  }

  static String _withoutTrailingSlash(String value) {
    return value.endsWith('/') ? value.substring(0, value.length - 1) : value;
  }
}

import 'package:dio/dio.dart';

import '../core/access_token.dart';
import '../data/database.dart';

class PullService {
  PullService({
    required this.database,
    required this.http,
    required this.tokens,
  });

  final MaintenanceDatabase database;
  final Dio http;
  final AccessTokenProvider tokens;

  Future<void> refreshAssignedExecutions() async {
    final token = await tokens.accessToken();
    if (token == null) return;
    final response = await http.get<Map<String, dynamic>>(
      '/api/v1/executions',
      queryParameters: const {'assigned_to': 'me'},
      options: Options(headers: {'Authorization': 'Bearer $token'}),
    );
    final items = response.data?['items'] as List<dynamic>? ?? const [];
    for (final raw in items) {
      await database.cacheExecution(raw as Map<String, dynamic>);
    }
  }
}

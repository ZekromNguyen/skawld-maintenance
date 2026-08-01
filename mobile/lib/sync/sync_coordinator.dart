import 'dart:async';

import 'package:connectivity_plus/connectivity_plus.dart';

import 'pull_service.dart';
import 'sync_engine.dart';

class SyncCoordinator {
  SyncCoordinator({
    required this.connectivity,
    required this.pull,
    required this.push,
  });

  final Connectivity connectivity;
  final PullService pull;
  final SyncEngine push;
  StreamSubscription<List<ConnectivityResult>>? _subscription;

  void start() {
    _subscription ??= connectivity.onConnectivityChanged.listen((states) {
      if (states.any((state) => state != ConnectivityResult.none)) {
        unawaited(synchronize());
      }
    });
  }

  Future<void> synchronize() async {
    await push.runOnce();
    await pull.refreshAssignedExecutions();
  }

  Future<void> dispose() async {
    await _subscription?.cancel();
    _subscription = null;
  }
}

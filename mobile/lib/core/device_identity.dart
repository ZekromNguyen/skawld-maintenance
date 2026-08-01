import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:uuid/uuid.dart';

class DeviceIdentity {
  DeviceIdentity({FlutterSecureStorage? storage, Uuid? uuid})
    : _storage = storage ?? const FlutterSecureStorage(),
      _uuid = uuid ?? const Uuid();

  static const _key = 'skawld.device_id';
  final FlutterSecureStorage _storage;
  final Uuid _uuid;

  Future<String> getOrCreate() async {
    final current = await _storage.read(key: _key);
    if (current != null && current.isNotEmpty) return current;
    final created = _uuid.v4();
    await _storage.write(key: _key, value: created);
    return created;
  }
}

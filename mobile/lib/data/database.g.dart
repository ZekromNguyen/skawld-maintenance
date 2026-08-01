// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'database.dart';

// ignore_for_file: type=lint
class $CachedAssetsTable extends CachedAssets
    with TableInfo<$CachedAssetsTable, CachedAsset> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $CachedAssetsTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _idMeta = const VerificationMeta('id');
  @override
  late final GeneratedColumn<String> id = GeneratedColumn<String>(
    'id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _siteIdMeta = const VerificationMeta('siteId');
  @override
  late final GeneratedColumn<String> siteId = GeneratedColumn<String>(
    'site_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _tagMeta = const VerificationMeta('tag');
  @override
  late final GeneratedColumn<String> tag = GeneratedColumn<String>(
    'tag',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _nameMeta = const VerificationMeta('name');
  @override
  late final GeneratedColumn<String> name = GeneratedColumn<String>(
    'name',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _assetClassMeta = const VerificationMeta(
    'assetClass',
  );
  @override
  late final GeneratedColumn<String> assetClass = GeneratedColumn<String>(
    'asset_class',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _sourceOfTruthMeta = const VerificationMeta(
    'sourceOfTruth',
  );
  @override
  late final GeneratedColumn<String> sourceOfTruth = GeneratedColumn<String>(
    'source_of_truth',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _criticalityMeta = const VerificationMeta(
    'criticality',
  );
  @override
  late final GeneratedColumn<String> criticality = GeneratedColumn<String>(
    'criticality',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _serverVersionMeta = const VerificationMeta(
    'serverVersion',
  );
  @override
  late final GeneratedColumn<int> serverVersion = GeneratedColumn<int>(
    'server_version',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _cachedAtMeta = const VerificationMeta(
    'cachedAt',
  );
  @override
  late final GeneratedColumn<DateTime> cachedAt = GeneratedColumn<DateTime>(
    'cached_at',
    aliasedName,
    false,
    type: DriftSqlType.dateTime,
    requiredDuringInsert: true,
  );
  @override
  List<GeneratedColumn> get $columns => [
    id,
    siteId,
    tag,
    name,
    assetClass,
    sourceOfTruth,
    criticality,
    serverVersion,
    cachedAt,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'cached_assets';
  @override
  VerificationContext validateIntegrity(
    Insertable<CachedAsset> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('id')) {
      context.handle(_idMeta, id.isAcceptableOrUnknown(data['id']!, _idMeta));
    } else if (isInserting) {
      context.missing(_idMeta);
    }
    if (data.containsKey('site_id')) {
      context.handle(
        _siteIdMeta,
        siteId.isAcceptableOrUnknown(data['site_id']!, _siteIdMeta),
      );
    } else if (isInserting) {
      context.missing(_siteIdMeta);
    }
    if (data.containsKey('tag')) {
      context.handle(
        _tagMeta,
        tag.isAcceptableOrUnknown(data['tag']!, _tagMeta),
      );
    } else if (isInserting) {
      context.missing(_tagMeta);
    }
    if (data.containsKey('name')) {
      context.handle(
        _nameMeta,
        name.isAcceptableOrUnknown(data['name']!, _nameMeta),
      );
    } else if (isInserting) {
      context.missing(_nameMeta);
    }
    if (data.containsKey('asset_class')) {
      context.handle(
        _assetClassMeta,
        assetClass.isAcceptableOrUnknown(data['asset_class']!, _assetClassMeta),
      );
    } else if (isInserting) {
      context.missing(_assetClassMeta);
    }
    if (data.containsKey('source_of_truth')) {
      context.handle(
        _sourceOfTruthMeta,
        sourceOfTruth.isAcceptableOrUnknown(
          data['source_of_truth']!,
          _sourceOfTruthMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_sourceOfTruthMeta);
    }
    if (data.containsKey('criticality')) {
      context.handle(
        _criticalityMeta,
        criticality.isAcceptableOrUnknown(
          data['criticality']!,
          _criticalityMeta,
        ),
      );
    }
    if (data.containsKey('server_version')) {
      context.handle(
        _serverVersionMeta,
        serverVersion.isAcceptableOrUnknown(
          data['server_version']!,
          _serverVersionMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_serverVersionMeta);
    }
    if (data.containsKey('cached_at')) {
      context.handle(
        _cachedAtMeta,
        cachedAt.isAcceptableOrUnknown(data['cached_at']!, _cachedAtMeta),
      );
    } else if (isInserting) {
      context.missing(_cachedAtMeta);
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {id};
  @override
  CachedAsset map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return CachedAsset(
      id: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}id'],
      )!,
      siteId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}site_id'],
      )!,
      tag: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}tag'],
      )!,
      name: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}name'],
      )!,
      assetClass: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}asset_class'],
      )!,
      sourceOfTruth: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}source_of_truth'],
      )!,
      criticality: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}criticality'],
      ),
      serverVersion: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}server_version'],
      )!,
      cachedAt: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}cached_at'],
      )!,
    );
  }

  @override
  $CachedAssetsTable createAlias(String alias) {
    return $CachedAssetsTable(attachedDatabase, alias);
  }
}

class CachedAsset extends DataClass implements Insertable<CachedAsset> {
  final String id;
  final String siteId;
  final String tag;
  final String name;
  final String assetClass;
  final String sourceOfTruth;
  final String? criticality;
  final int serverVersion;
  final DateTime cachedAt;
  const CachedAsset({
    required this.id,
    required this.siteId,
    required this.tag,
    required this.name,
    required this.assetClass,
    required this.sourceOfTruth,
    this.criticality,
    required this.serverVersion,
    required this.cachedAt,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['id'] = Variable<String>(id);
    map['site_id'] = Variable<String>(siteId);
    map['tag'] = Variable<String>(tag);
    map['name'] = Variable<String>(name);
    map['asset_class'] = Variable<String>(assetClass);
    map['source_of_truth'] = Variable<String>(sourceOfTruth);
    if (!nullToAbsent || criticality != null) {
      map['criticality'] = Variable<String>(criticality);
    }
    map['server_version'] = Variable<int>(serverVersion);
    map['cached_at'] = Variable<DateTime>(cachedAt);
    return map;
  }

  CachedAssetsCompanion toCompanion(bool nullToAbsent) {
    return CachedAssetsCompanion(
      id: Value(id),
      siteId: Value(siteId),
      tag: Value(tag),
      name: Value(name),
      assetClass: Value(assetClass),
      sourceOfTruth: Value(sourceOfTruth),
      criticality: criticality == null && nullToAbsent
          ? const Value.absent()
          : Value(criticality),
      serverVersion: Value(serverVersion),
      cachedAt: Value(cachedAt),
    );
  }

  factory CachedAsset.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return CachedAsset(
      id: serializer.fromJson<String>(json['id']),
      siteId: serializer.fromJson<String>(json['siteId']),
      tag: serializer.fromJson<String>(json['tag']),
      name: serializer.fromJson<String>(json['name']),
      assetClass: serializer.fromJson<String>(json['assetClass']),
      sourceOfTruth: serializer.fromJson<String>(json['sourceOfTruth']),
      criticality: serializer.fromJson<String?>(json['criticality']),
      serverVersion: serializer.fromJson<int>(json['serverVersion']),
      cachedAt: serializer.fromJson<DateTime>(json['cachedAt']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'id': serializer.toJson<String>(id),
      'siteId': serializer.toJson<String>(siteId),
      'tag': serializer.toJson<String>(tag),
      'name': serializer.toJson<String>(name),
      'assetClass': serializer.toJson<String>(assetClass),
      'sourceOfTruth': serializer.toJson<String>(sourceOfTruth),
      'criticality': serializer.toJson<String?>(criticality),
      'serverVersion': serializer.toJson<int>(serverVersion),
      'cachedAt': serializer.toJson<DateTime>(cachedAt),
    };
  }

  CachedAsset copyWith({
    String? id,
    String? siteId,
    String? tag,
    String? name,
    String? assetClass,
    String? sourceOfTruth,
    Value<String?> criticality = const Value.absent(),
    int? serverVersion,
    DateTime? cachedAt,
  }) => CachedAsset(
    id: id ?? this.id,
    siteId: siteId ?? this.siteId,
    tag: tag ?? this.tag,
    name: name ?? this.name,
    assetClass: assetClass ?? this.assetClass,
    sourceOfTruth: sourceOfTruth ?? this.sourceOfTruth,
    criticality: criticality.present ? criticality.value : this.criticality,
    serverVersion: serverVersion ?? this.serverVersion,
    cachedAt: cachedAt ?? this.cachedAt,
  );
  CachedAsset copyWithCompanion(CachedAssetsCompanion data) {
    return CachedAsset(
      id: data.id.present ? data.id.value : this.id,
      siteId: data.siteId.present ? data.siteId.value : this.siteId,
      tag: data.tag.present ? data.tag.value : this.tag,
      name: data.name.present ? data.name.value : this.name,
      assetClass: data.assetClass.present
          ? data.assetClass.value
          : this.assetClass,
      sourceOfTruth: data.sourceOfTruth.present
          ? data.sourceOfTruth.value
          : this.sourceOfTruth,
      criticality: data.criticality.present
          ? data.criticality.value
          : this.criticality,
      serverVersion: data.serverVersion.present
          ? data.serverVersion.value
          : this.serverVersion,
      cachedAt: data.cachedAt.present ? data.cachedAt.value : this.cachedAt,
    );
  }

  @override
  String toString() {
    return (StringBuffer('CachedAsset(')
          ..write('id: $id, ')
          ..write('siteId: $siteId, ')
          ..write('tag: $tag, ')
          ..write('name: $name, ')
          ..write('assetClass: $assetClass, ')
          ..write('sourceOfTruth: $sourceOfTruth, ')
          ..write('criticality: $criticality, ')
          ..write('serverVersion: $serverVersion, ')
          ..write('cachedAt: $cachedAt')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(
    id,
    siteId,
    tag,
    name,
    assetClass,
    sourceOfTruth,
    criticality,
    serverVersion,
    cachedAt,
  );
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is CachedAsset &&
          other.id == this.id &&
          other.siteId == this.siteId &&
          other.tag == this.tag &&
          other.name == this.name &&
          other.assetClass == this.assetClass &&
          other.sourceOfTruth == this.sourceOfTruth &&
          other.criticality == this.criticality &&
          other.serverVersion == this.serverVersion &&
          other.cachedAt == this.cachedAt);
}

class CachedAssetsCompanion extends UpdateCompanion<CachedAsset> {
  final Value<String> id;
  final Value<String> siteId;
  final Value<String> tag;
  final Value<String> name;
  final Value<String> assetClass;
  final Value<String> sourceOfTruth;
  final Value<String?> criticality;
  final Value<int> serverVersion;
  final Value<DateTime> cachedAt;
  final Value<int> rowid;
  const CachedAssetsCompanion({
    this.id = const Value.absent(),
    this.siteId = const Value.absent(),
    this.tag = const Value.absent(),
    this.name = const Value.absent(),
    this.assetClass = const Value.absent(),
    this.sourceOfTruth = const Value.absent(),
    this.criticality = const Value.absent(),
    this.serverVersion = const Value.absent(),
    this.cachedAt = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  CachedAssetsCompanion.insert({
    required String id,
    required String siteId,
    required String tag,
    required String name,
    required String assetClass,
    required String sourceOfTruth,
    this.criticality = const Value.absent(),
    required int serverVersion,
    required DateTime cachedAt,
    this.rowid = const Value.absent(),
  }) : id = Value(id),
       siteId = Value(siteId),
       tag = Value(tag),
       name = Value(name),
       assetClass = Value(assetClass),
       sourceOfTruth = Value(sourceOfTruth),
       serverVersion = Value(serverVersion),
       cachedAt = Value(cachedAt);
  static Insertable<CachedAsset> custom({
    Expression<String>? id,
    Expression<String>? siteId,
    Expression<String>? tag,
    Expression<String>? name,
    Expression<String>? assetClass,
    Expression<String>? sourceOfTruth,
    Expression<String>? criticality,
    Expression<int>? serverVersion,
    Expression<DateTime>? cachedAt,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (id != null) 'id': id,
      if (siteId != null) 'site_id': siteId,
      if (tag != null) 'tag': tag,
      if (name != null) 'name': name,
      if (assetClass != null) 'asset_class': assetClass,
      if (sourceOfTruth != null) 'source_of_truth': sourceOfTruth,
      if (criticality != null) 'criticality': criticality,
      if (serverVersion != null) 'server_version': serverVersion,
      if (cachedAt != null) 'cached_at': cachedAt,
      if (rowid != null) 'rowid': rowid,
    });
  }

  CachedAssetsCompanion copyWith({
    Value<String>? id,
    Value<String>? siteId,
    Value<String>? tag,
    Value<String>? name,
    Value<String>? assetClass,
    Value<String>? sourceOfTruth,
    Value<String?>? criticality,
    Value<int>? serverVersion,
    Value<DateTime>? cachedAt,
    Value<int>? rowid,
  }) {
    return CachedAssetsCompanion(
      id: id ?? this.id,
      siteId: siteId ?? this.siteId,
      tag: tag ?? this.tag,
      name: name ?? this.name,
      assetClass: assetClass ?? this.assetClass,
      sourceOfTruth: sourceOfTruth ?? this.sourceOfTruth,
      criticality: criticality ?? this.criticality,
      serverVersion: serverVersion ?? this.serverVersion,
      cachedAt: cachedAt ?? this.cachedAt,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (id.present) {
      map['id'] = Variable<String>(id.value);
    }
    if (siteId.present) {
      map['site_id'] = Variable<String>(siteId.value);
    }
    if (tag.present) {
      map['tag'] = Variable<String>(tag.value);
    }
    if (name.present) {
      map['name'] = Variable<String>(name.value);
    }
    if (assetClass.present) {
      map['asset_class'] = Variable<String>(assetClass.value);
    }
    if (sourceOfTruth.present) {
      map['source_of_truth'] = Variable<String>(sourceOfTruth.value);
    }
    if (criticality.present) {
      map['criticality'] = Variable<String>(criticality.value);
    }
    if (serverVersion.present) {
      map['server_version'] = Variable<int>(serverVersion.value);
    }
    if (cachedAt.present) {
      map['cached_at'] = Variable<DateTime>(cachedAt.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('CachedAssetsCompanion(')
          ..write('id: $id, ')
          ..write('siteId: $siteId, ')
          ..write('tag: $tag, ')
          ..write('name: $name, ')
          ..write('assetClass: $assetClass, ')
          ..write('sourceOfTruth: $sourceOfTruth, ')
          ..write('criticality: $criticality, ')
          ..write('serverVersion: $serverVersion, ')
          ..write('cachedAt: $cachedAt, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $CachedIncidentsTable extends CachedIncidents
    with TableInfo<$CachedIncidentsTable, CachedIncident> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $CachedIncidentsTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _idMeta = const VerificationMeta('id');
  @override
  late final GeneratedColumn<String> id = GeneratedColumn<String>(
    'id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _siteIdMeta = const VerificationMeta('siteId');
  @override
  late final GeneratedColumn<String> siteId = GeneratedColumn<String>(
    'site_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _assetIdMeta = const VerificationMeta(
    'assetId',
  );
  @override
  late final GeneratedColumn<String> assetId = GeneratedColumn<String>(
    'asset_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _assetTagMeta = const VerificationMeta(
    'assetTag',
  );
  @override
  late final GeneratedColumn<String> assetTag = GeneratedColumn<String>(
    'asset_tag',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _numberMeta = const VerificationMeta('number');
  @override
  late final GeneratedColumn<String> number = GeneratedColumn<String>(
    'number',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _summaryMeta = const VerificationMeta(
    'summary',
  );
  @override
  late final GeneratedColumn<String> summary = GeneratedColumn<String>(
    'summary',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _severityMeta = const VerificationMeta(
    'severity',
  );
  @override
  late final GeneratedColumn<String> severity = GeneratedColumn<String>(
    'severity',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _stateMeta = const VerificationMeta('state');
  @override
  late final GeneratedColumn<String> state = GeneratedColumn<String>(
    'state',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _detectedAtMeta = const VerificationMeta(
    'detectedAt',
  );
  @override
  late final GeneratedColumn<DateTime> detectedAt = GeneratedColumn<DateTime>(
    'detected_at',
    aliasedName,
    false,
    type: DriftSqlType.dateTime,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _serverVersionMeta = const VerificationMeta(
    'serverVersion',
  );
  @override
  late final GeneratedColumn<int> serverVersion = GeneratedColumn<int>(
    'server_version',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _cachedAtMeta = const VerificationMeta(
    'cachedAt',
  );
  @override
  late final GeneratedColumn<DateTime> cachedAt = GeneratedColumn<DateTime>(
    'cached_at',
    aliasedName,
    false,
    type: DriftSqlType.dateTime,
    requiredDuringInsert: true,
  );
  @override
  List<GeneratedColumn> get $columns => [
    id,
    siteId,
    assetId,
    assetTag,
    number,
    summary,
    severity,
    state,
    detectedAt,
    serverVersion,
    cachedAt,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'cached_incidents';
  @override
  VerificationContext validateIntegrity(
    Insertable<CachedIncident> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('id')) {
      context.handle(_idMeta, id.isAcceptableOrUnknown(data['id']!, _idMeta));
    } else if (isInserting) {
      context.missing(_idMeta);
    }
    if (data.containsKey('site_id')) {
      context.handle(
        _siteIdMeta,
        siteId.isAcceptableOrUnknown(data['site_id']!, _siteIdMeta),
      );
    } else if (isInserting) {
      context.missing(_siteIdMeta);
    }
    if (data.containsKey('asset_id')) {
      context.handle(
        _assetIdMeta,
        assetId.isAcceptableOrUnknown(data['asset_id']!, _assetIdMeta),
      );
    } else if (isInserting) {
      context.missing(_assetIdMeta);
    }
    if (data.containsKey('asset_tag')) {
      context.handle(
        _assetTagMeta,
        assetTag.isAcceptableOrUnknown(data['asset_tag']!, _assetTagMeta),
      );
    } else if (isInserting) {
      context.missing(_assetTagMeta);
    }
    if (data.containsKey('number')) {
      context.handle(
        _numberMeta,
        number.isAcceptableOrUnknown(data['number']!, _numberMeta),
      );
    } else if (isInserting) {
      context.missing(_numberMeta);
    }
    if (data.containsKey('summary')) {
      context.handle(
        _summaryMeta,
        summary.isAcceptableOrUnknown(data['summary']!, _summaryMeta),
      );
    } else if (isInserting) {
      context.missing(_summaryMeta);
    }
    if (data.containsKey('severity')) {
      context.handle(
        _severityMeta,
        severity.isAcceptableOrUnknown(data['severity']!, _severityMeta),
      );
    } else if (isInserting) {
      context.missing(_severityMeta);
    }
    if (data.containsKey('state')) {
      context.handle(
        _stateMeta,
        state.isAcceptableOrUnknown(data['state']!, _stateMeta),
      );
    } else if (isInserting) {
      context.missing(_stateMeta);
    }
    if (data.containsKey('detected_at')) {
      context.handle(
        _detectedAtMeta,
        detectedAt.isAcceptableOrUnknown(data['detected_at']!, _detectedAtMeta),
      );
    } else if (isInserting) {
      context.missing(_detectedAtMeta);
    }
    if (data.containsKey('server_version')) {
      context.handle(
        _serverVersionMeta,
        serverVersion.isAcceptableOrUnknown(
          data['server_version']!,
          _serverVersionMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_serverVersionMeta);
    }
    if (data.containsKey('cached_at')) {
      context.handle(
        _cachedAtMeta,
        cachedAt.isAcceptableOrUnknown(data['cached_at']!, _cachedAtMeta),
      );
    } else if (isInserting) {
      context.missing(_cachedAtMeta);
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {id};
  @override
  CachedIncident map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return CachedIncident(
      id: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}id'],
      )!,
      siteId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}site_id'],
      )!,
      assetId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}asset_id'],
      )!,
      assetTag: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}asset_tag'],
      )!,
      number: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}number'],
      )!,
      summary: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}summary'],
      )!,
      severity: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}severity'],
      )!,
      state: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}state'],
      )!,
      detectedAt: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}detected_at'],
      )!,
      serverVersion: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}server_version'],
      )!,
      cachedAt: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}cached_at'],
      )!,
    );
  }

  @override
  $CachedIncidentsTable createAlias(String alias) {
    return $CachedIncidentsTable(attachedDatabase, alias);
  }
}

class CachedIncident extends DataClass implements Insertable<CachedIncident> {
  final String id;
  final String siteId;
  final String assetId;
  final String assetTag;
  final String number;
  final String summary;
  final String severity;
  final String state;
  final DateTime detectedAt;
  final int serverVersion;
  final DateTime cachedAt;
  const CachedIncident({
    required this.id,
    required this.siteId,
    required this.assetId,
    required this.assetTag,
    required this.number,
    required this.summary,
    required this.severity,
    required this.state,
    required this.detectedAt,
    required this.serverVersion,
    required this.cachedAt,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['id'] = Variable<String>(id);
    map['site_id'] = Variable<String>(siteId);
    map['asset_id'] = Variable<String>(assetId);
    map['asset_tag'] = Variable<String>(assetTag);
    map['number'] = Variable<String>(number);
    map['summary'] = Variable<String>(summary);
    map['severity'] = Variable<String>(severity);
    map['state'] = Variable<String>(state);
    map['detected_at'] = Variable<DateTime>(detectedAt);
    map['server_version'] = Variable<int>(serverVersion);
    map['cached_at'] = Variable<DateTime>(cachedAt);
    return map;
  }

  CachedIncidentsCompanion toCompanion(bool nullToAbsent) {
    return CachedIncidentsCompanion(
      id: Value(id),
      siteId: Value(siteId),
      assetId: Value(assetId),
      assetTag: Value(assetTag),
      number: Value(number),
      summary: Value(summary),
      severity: Value(severity),
      state: Value(state),
      detectedAt: Value(detectedAt),
      serverVersion: Value(serverVersion),
      cachedAt: Value(cachedAt),
    );
  }

  factory CachedIncident.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return CachedIncident(
      id: serializer.fromJson<String>(json['id']),
      siteId: serializer.fromJson<String>(json['siteId']),
      assetId: serializer.fromJson<String>(json['assetId']),
      assetTag: serializer.fromJson<String>(json['assetTag']),
      number: serializer.fromJson<String>(json['number']),
      summary: serializer.fromJson<String>(json['summary']),
      severity: serializer.fromJson<String>(json['severity']),
      state: serializer.fromJson<String>(json['state']),
      detectedAt: serializer.fromJson<DateTime>(json['detectedAt']),
      serverVersion: serializer.fromJson<int>(json['serverVersion']),
      cachedAt: serializer.fromJson<DateTime>(json['cachedAt']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'id': serializer.toJson<String>(id),
      'siteId': serializer.toJson<String>(siteId),
      'assetId': serializer.toJson<String>(assetId),
      'assetTag': serializer.toJson<String>(assetTag),
      'number': serializer.toJson<String>(number),
      'summary': serializer.toJson<String>(summary),
      'severity': serializer.toJson<String>(severity),
      'state': serializer.toJson<String>(state),
      'detectedAt': serializer.toJson<DateTime>(detectedAt),
      'serverVersion': serializer.toJson<int>(serverVersion),
      'cachedAt': serializer.toJson<DateTime>(cachedAt),
    };
  }

  CachedIncident copyWith({
    String? id,
    String? siteId,
    String? assetId,
    String? assetTag,
    String? number,
    String? summary,
    String? severity,
    String? state,
    DateTime? detectedAt,
    int? serverVersion,
    DateTime? cachedAt,
  }) => CachedIncident(
    id: id ?? this.id,
    siteId: siteId ?? this.siteId,
    assetId: assetId ?? this.assetId,
    assetTag: assetTag ?? this.assetTag,
    number: number ?? this.number,
    summary: summary ?? this.summary,
    severity: severity ?? this.severity,
    state: state ?? this.state,
    detectedAt: detectedAt ?? this.detectedAt,
    serverVersion: serverVersion ?? this.serverVersion,
    cachedAt: cachedAt ?? this.cachedAt,
  );
  CachedIncident copyWithCompanion(CachedIncidentsCompanion data) {
    return CachedIncident(
      id: data.id.present ? data.id.value : this.id,
      siteId: data.siteId.present ? data.siteId.value : this.siteId,
      assetId: data.assetId.present ? data.assetId.value : this.assetId,
      assetTag: data.assetTag.present ? data.assetTag.value : this.assetTag,
      number: data.number.present ? data.number.value : this.number,
      summary: data.summary.present ? data.summary.value : this.summary,
      severity: data.severity.present ? data.severity.value : this.severity,
      state: data.state.present ? data.state.value : this.state,
      detectedAt: data.detectedAt.present
          ? data.detectedAt.value
          : this.detectedAt,
      serverVersion: data.serverVersion.present
          ? data.serverVersion.value
          : this.serverVersion,
      cachedAt: data.cachedAt.present ? data.cachedAt.value : this.cachedAt,
    );
  }

  @override
  String toString() {
    return (StringBuffer('CachedIncident(')
          ..write('id: $id, ')
          ..write('siteId: $siteId, ')
          ..write('assetId: $assetId, ')
          ..write('assetTag: $assetTag, ')
          ..write('number: $number, ')
          ..write('summary: $summary, ')
          ..write('severity: $severity, ')
          ..write('state: $state, ')
          ..write('detectedAt: $detectedAt, ')
          ..write('serverVersion: $serverVersion, ')
          ..write('cachedAt: $cachedAt')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(
    id,
    siteId,
    assetId,
    assetTag,
    number,
    summary,
    severity,
    state,
    detectedAt,
    serverVersion,
    cachedAt,
  );
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is CachedIncident &&
          other.id == this.id &&
          other.siteId == this.siteId &&
          other.assetId == this.assetId &&
          other.assetTag == this.assetTag &&
          other.number == this.number &&
          other.summary == this.summary &&
          other.severity == this.severity &&
          other.state == this.state &&
          other.detectedAt == this.detectedAt &&
          other.serverVersion == this.serverVersion &&
          other.cachedAt == this.cachedAt);
}

class CachedIncidentsCompanion extends UpdateCompanion<CachedIncident> {
  final Value<String> id;
  final Value<String> siteId;
  final Value<String> assetId;
  final Value<String> assetTag;
  final Value<String> number;
  final Value<String> summary;
  final Value<String> severity;
  final Value<String> state;
  final Value<DateTime> detectedAt;
  final Value<int> serverVersion;
  final Value<DateTime> cachedAt;
  final Value<int> rowid;
  const CachedIncidentsCompanion({
    this.id = const Value.absent(),
    this.siteId = const Value.absent(),
    this.assetId = const Value.absent(),
    this.assetTag = const Value.absent(),
    this.number = const Value.absent(),
    this.summary = const Value.absent(),
    this.severity = const Value.absent(),
    this.state = const Value.absent(),
    this.detectedAt = const Value.absent(),
    this.serverVersion = const Value.absent(),
    this.cachedAt = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  CachedIncidentsCompanion.insert({
    required String id,
    required String siteId,
    required String assetId,
    required String assetTag,
    required String number,
    required String summary,
    required String severity,
    required String state,
    required DateTime detectedAt,
    required int serverVersion,
    required DateTime cachedAt,
    this.rowid = const Value.absent(),
  }) : id = Value(id),
       siteId = Value(siteId),
       assetId = Value(assetId),
       assetTag = Value(assetTag),
       number = Value(number),
       summary = Value(summary),
       severity = Value(severity),
       state = Value(state),
       detectedAt = Value(detectedAt),
       serverVersion = Value(serverVersion),
       cachedAt = Value(cachedAt);
  static Insertable<CachedIncident> custom({
    Expression<String>? id,
    Expression<String>? siteId,
    Expression<String>? assetId,
    Expression<String>? assetTag,
    Expression<String>? number,
    Expression<String>? summary,
    Expression<String>? severity,
    Expression<String>? state,
    Expression<DateTime>? detectedAt,
    Expression<int>? serverVersion,
    Expression<DateTime>? cachedAt,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (id != null) 'id': id,
      if (siteId != null) 'site_id': siteId,
      if (assetId != null) 'asset_id': assetId,
      if (assetTag != null) 'asset_tag': assetTag,
      if (number != null) 'number': number,
      if (summary != null) 'summary': summary,
      if (severity != null) 'severity': severity,
      if (state != null) 'state': state,
      if (detectedAt != null) 'detected_at': detectedAt,
      if (serverVersion != null) 'server_version': serverVersion,
      if (cachedAt != null) 'cached_at': cachedAt,
      if (rowid != null) 'rowid': rowid,
    });
  }

  CachedIncidentsCompanion copyWith({
    Value<String>? id,
    Value<String>? siteId,
    Value<String>? assetId,
    Value<String>? assetTag,
    Value<String>? number,
    Value<String>? summary,
    Value<String>? severity,
    Value<String>? state,
    Value<DateTime>? detectedAt,
    Value<int>? serverVersion,
    Value<DateTime>? cachedAt,
    Value<int>? rowid,
  }) {
    return CachedIncidentsCompanion(
      id: id ?? this.id,
      siteId: siteId ?? this.siteId,
      assetId: assetId ?? this.assetId,
      assetTag: assetTag ?? this.assetTag,
      number: number ?? this.number,
      summary: summary ?? this.summary,
      severity: severity ?? this.severity,
      state: state ?? this.state,
      detectedAt: detectedAt ?? this.detectedAt,
      serverVersion: serverVersion ?? this.serverVersion,
      cachedAt: cachedAt ?? this.cachedAt,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (id.present) {
      map['id'] = Variable<String>(id.value);
    }
    if (siteId.present) {
      map['site_id'] = Variable<String>(siteId.value);
    }
    if (assetId.present) {
      map['asset_id'] = Variable<String>(assetId.value);
    }
    if (assetTag.present) {
      map['asset_tag'] = Variable<String>(assetTag.value);
    }
    if (number.present) {
      map['number'] = Variable<String>(number.value);
    }
    if (summary.present) {
      map['summary'] = Variable<String>(summary.value);
    }
    if (severity.present) {
      map['severity'] = Variable<String>(severity.value);
    }
    if (state.present) {
      map['state'] = Variable<String>(state.value);
    }
    if (detectedAt.present) {
      map['detected_at'] = Variable<DateTime>(detectedAt.value);
    }
    if (serverVersion.present) {
      map['server_version'] = Variable<int>(serverVersion.value);
    }
    if (cachedAt.present) {
      map['cached_at'] = Variable<DateTime>(cachedAt.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('CachedIncidentsCompanion(')
          ..write('id: $id, ')
          ..write('siteId: $siteId, ')
          ..write('assetId: $assetId, ')
          ..write('assetTag: $assetTag, ')
          ..write('number: $number, ')
          ..write('summary: $summary, ')
          ..write('severity: $severity, ')
          ..write('state: $state, ')
          ..write('detectedAt: $detectedAt, ')
          ..write('serverVersion: $serverVersion, ')
          ..write('cachedAt: $cachedAt, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $CachedExecutionsTable extends CachedExecutions
    with TableInfo<$CachedExecutionsTable, CachedExecution> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $CachedExecutionsTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _idMeta = const VerificationMeta('id');
  @override
  late final GeneratedColumn<String> id = GeneratedColumn<String>(
    'id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _siteIdMeta = const VerificationMeta('siteId');
  @override
  late final GeneratedColumn<String> siteId = GeneratedColumn<String>(
    'site_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _incidentIdMeta = const VerificationMeta(
    'incidentId',
  );
  @override
  late final GeneratedColumn<String> incidentId = GeneratedColumn<String>(
    'incident_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _assetIdMeta = const VerificationMeta(
    'assetId',
  );
  @override
  late final GeneratedColumn<String> assetId = GeneratedColumn<String>(
    'asset_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _assetTagMeta = const VerificationMeta(
    'assetTag',
  );
  @override
  late final GeneratedColumn<String> assetTag = GeneratedColumn<String>(
    'asset_tag',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _purposeMeta = const VerificationMeta(
    'purpose',
  );
  @override
  late final GeneratedColumn<String> purpose = GeneratedColumn<String>(
    'purpose',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _stateMeta = const VerificationMeta('state');
  @override
  late final GeneratedColumn<String> state = GeneratedColumn<String>(
    'state',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _serverVersionMeta = const VerificationMeta(
    'serverVersion',
  );
  @override
  late final GeneratedColumn<int> serverVersion = GeneratedColumn<int>(
    'server_version',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _verifiedPrerequisitesMeta =
      const VerificationMeta('verifiedPrerequisites');
  @override
  late final GeneratedColumn<String> verifiedPrerequisites =
      GeneratedColumn<String>(
        'verified_prerequisites',
        aliasedName,
        false,
        type: DriftSqlType.string,
        requiredDuringInsert: false,
        defaultValue: const Constant(''),
      );
  static const VerificationMeta _cachedAtMeta = const VerificationMeta(
    'cachedAt',
  );
  @override
  late final GeneratedColumn<DateTime> cachedAt = GeneratedColumn<DateTime>(
    'cached_at',
    aliasedName,
    false,
    type: DriftSqlType.dateTime,
    requiredDuringInsert: true,
  );
  @override
  List<GeneratedColumn> get $columns => [
    id,
    siteId,
    incidentId,
    assetId,
    assetTag,
    purpose,
    state,
    serverVersion,
    verifiedPrerequisites,
    cachedAt,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'cached_executions';
  @override
  VerificationContext validateIntegrity(
    Insertable<CachedExecution> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('id')) {
      context.handle(_idMeta, id.isAcceptableOrUnknown(data['id']!, _idMeta));
    } else if (isInserting) {
      context.missing(_idMeta);
    }
    if (data.containsKey('site_id')) {
      context.handle(
        _siteIdMeta,
        siteId.isAcceptableOrUnknown(data['site_id']!, _siteIdMeta),
      );
    } else if (isInserting) {
      context.missing(_siteIdMeta);
    }
    if (data.containsKey('incident_id')) {
      context.handle(
        _incidentIdMeta,
        incidentId.isAcceptableOrUnknown(data['incident_id']!, _incidentIdMeta),
      );
    } else if (isInserting) {
      context.missing(_incidentIdMeta);
    }
    if (data.containsKey('asset_id')) {
      context.handle(
        _assetIdMeta,
        assetId.isAcceptableOrUnknown(data['asset_id']!, _assetIdMeta),
      );
    } else if (isInserting) {
      context.missing(_assetIdMeta);
    }
    if (data.containsKey('asset_tag')) {
      context.handle(
        _assetTagMeta,
        assetTag.isAcceptableOrUnknown(data['asset_tag']!, _assetTagMeta),
      );
    } else if (isInserting) {
      context.missing(_assetTagMeta);
    }
    if (data.containsKey('purpose')) {
      context.handle(
        _purposeMeta,
        purpose.isAcceptableOrUnknown(data['purpose']!, _purposeMeta),
      );
    } else if (isInserting) {
      context.missing(_purposeMeta);
    }
    if (data.containsKey('state')) {
      context.handle(
        _stateMeta,
        state.isAcceptableOrUnknown(data['state']!, _stateMeta),
      );
    } else if (isInserting) {
      context.missing(_stateMeta);
    }
    if (data.containsKey('server_version')) {
      context.handle(
        _serverVersionMeta,
        serverVersion.isAcceptableOrUnknown(
          data['server_version']!,
          _serverVersionMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_serverVersionMeta);
    }
    if (data.containsKey('verified_prerequisites')) {
      context.handle(
        _verifiedPrerequisitesMeta,
        verifiedPrerequisites.isAcceptableOrUnknown(
          data['verified_prerequisites']!,
          _verifiedPrerequisitesMeta,
        ),
      );
    }
    if (data.containsKey('cached_at')) {
      context.handle(
        _cachedAtMeta,
        cachedAt.isAcceptableOrUnknown(data['cached_at']!, _cachedAtMeta),
      );
    } else if (isInserting) {
      context.missing(_cachedAtMeta);
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {id};
  @override
  CachedExecution map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return CachedExecution(
      id: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}id'],
      )!,
      siteId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}site_id'],
      )!,
      incidentId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}incident_id'],
      )!,
      assetId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}asset_id'],
      )!,
      assetTag: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}asset_tag'],
      )!,
      purpose: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}purpose'],
      )!,
      state: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}state'],
      )!,
      serverVersion: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}server_version'],
      )!,
      verifiedPrerequisites: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}verified_prerequisites'],
      )!,
      cachedAt: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}cached_at'],
      )!,
    );
  }

  @override
  $CachedExecutionsTable createAlias(String alias) {
    return $CachedExecutionsTable(attachedDatabase, alias);
  }
}

class CachedExecution extends DataClass implements Insertable<CachedExecution> {
  final String id;
  final String siteId;
  final String incidentId;
  final String assetId;
  final String assetTag;
  final String purpose;
  final String state;
  final int serverVersion;
  final String verifiedPrerequisites;
  final DateTime cachedAt;
  const CachedExecution({
    required this.id,
    required this.siteId,
    required this.incidentId,
    required this.assetId,
    required this.assetTag,
    required this.purpose,
    required this.state,
    required this.serverVersion,
    required this.verifiedPrerequisites,
    required this.cachedAt,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['id'] = Variable<String>(id);
    map['site_id'] = Variable<String>(siteId);
    map['incident_id'] = Variable<String>(incidentId);
    map['asset_id'] = Variable<String>(assetId);
    map['asset_tag'] = Variable<String>(assetTag);
    map['purpose'] = Variable<String>(purpose);
    map['state'] = Variable<String>(state);
    map['server_version'] = Variable<int>(serverVersion);
    map['verified_prerequisites'] = Variable<String>(verifiedPrerequisites);
    map['cached_at'] = Variable<DateTime>(cachedAt);
    return map;
  }

  CachedExecutionsCompanion toCompanion(bool nullToAbsent) {
    return CachedExecutionsCompanion(
      id: Value(id),
      siteId: Value(siteId),
      incidentId: Value(incidentId),
      assetId: Value(assetId),
      assetTag: Value(assetTag),
      purpose: Value(purpose),
      state: Value(state),
      serverVersion: Value(serverVersion),
      verifiedPrerequisites: Value(verifiedPrerequisites),
      cachedAt: Value(cachedAt),
    );
  }

  factory CachedExecution.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return CachedExecution(
      id: serializer.fromJson<String>(json['id']),
      siteId: serializer.fromJson<String>(json['siteId']),
      incidentId: serializer.fromJson<String>(json['incidentId']),
      assetId: serializer.fromJson<String>(json['assetId']),
      assetTag: serializer.fromJson<String>(json['assetTag']),
      purpose: serializer.fromJson<String>(json['purpose']),
      state: serializer.fromJson<String>(json['state']),
      serverVersion: serializer.fromJson<int>(json['serverVersion']),
      verifiedPrerequisites: serializer.fromJson<String>(
        json['verifiedPrerequisites'],
      ),
      cachedAt: serializer.fromJson<DateTime>(json['cachedAt']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'id': serializer.toJson<String>(id),
      'siteId': serializer.toJson<String>(siteId),
      'incidentId': serializer.toJson<String>(incidentId),
      'assetId': serializer.toJson<String>(assetId),
      'assetTag': serializer.toJson<String>(assetTag),
      'purpose': serializer.toJson<String>(purpose),
      'state': serializer.toJson<String>(state),
      'serverVersion': serializer.toJson<int>(serverVersion),
      'verifiedPrerequisites': serializer.toJson<String>(verifiedPrerequisites),
      'cachedAt': serializer.toJson<DateTime>(cachedAt),
    };
  }

  CachedExecution copyWith({
    String? id,
    String? siteId,
    String? incidentId,
    String? assetId,
    String? assetTag,
    String? purpose,
    String? state,
    int? serverVersion,
    String? verifiedPrerequisites,
    DateTime? cachedAt,
  }) => CachedExecution(
    id: id ?? this.id,
    siteId: siteId ?? this.siteId,
    incidentId: incidentId ?? this.incidentId,
    assetId: assetId ?? this.assetId,
    assetTag: assetTag ?? this.assetTag,
    purpose: purpose ?? this.purpose,
    state: state ?? this.state,
    serverVersion: serverVersion ?? this.serverVersion,
    verifiedPrerequisites: verifiedPrerequisites ?? this.verifiedPrerequisites,
    cachedAt: cachedAt ?? this.cachedAt,
  );
  CachedExecution copyWithCompanion(CachedExecutionsCompanion data) {
    return CachedExecution(
      id: data.id.present ? data.id.value : this.id,
      siteId: data.siteId.present ? data.siteId.value : this.siteId,
      incidentId: data.incidentId.present
          ? data.incidentId.value
          : this.incidentId,
      assetId: data.assetId.present ? data.assetId.value : this.assetId,
      assetTag: data.assetTag.present ? data.assetTag.value : this.assetTag,
      purpose: data.purpose.present ? data.purpose.value : this.purpose,
      state: data.state.present ? data.state.value : this.state,
      serverVersion: data.serverVersion.present
          ? data.serverVersion.value
          : this.serverVersion,
      verifiedPrerequisites: data.verifiedPrerequisites.present
          ? data.verifiedPrerequisites.value
          : this.verifiedPrerequisites,
      cachedAt: data.cachedAt.present ? data.cachedAt.value : this.cachedAt,
    );
  }

  @override
  String toString() {
    return (StringBuffer('CachedExecution(')
          ..write('id: $id, ')
          ..write('siteId: $siteId, ')
          ..write('incidentId: $incidentId, ')
          ..write('assetId: $assetId, ')
          ..write('assetTag: $assetTag, ')
          ..write('purpose: $purpose, ')
          ..write('state: $state, ')
          ..write('serverVersion: $serverVersion, ')
          ..write('verifiedPrerequisites: $verifiedPrerequisites, ')
          ..write('cachedAt: $cachedAt')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(
    id,
    siteId,
    incidentId,
    assetId,
    assetTag,
    purpose,
    state,
    serverVersion,
    verifiedPrerequisites,
    cachedAt,
  );
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is CachedExecution &&
          other.id == this.id &&
          other.siteId == this.siteId &&
          other.incidentId == this.incidentId &&
          other.assetId == this.assetId &&
          other.assetTag == this.assetTag &&
          other.purpose == this.purpose &&
          other.state == this.state &&
          other.serverVersion == this.serverVersion &&
          other.verifiedPrerequisites == this.verifiedPrerequisites &&
          other.cachedAt == this.cachedAt);
}

class CachedExecutionsCompanion extends UpdateCompanion<CachedExecution> {
  final Value<String> id;
  final Value<String> siteId;
  final Value<String> incidentId;
  final Value<String> assetId;
  final Value<String> assetTag;
  final Value<String> purpose;
  final Value<String> state;
  final Value<int> serverVersion;
  final Value<String> verifiedPrerequisites;
  final Value<DateTime> cachedAt;
  final Value<int> rowid;
  const CachedExecutionsCompanion({
    this.id = const Value.absent(),
    this.siteId = const Value.absent(),
    this.incidentId = const Value.absent(),
    this.assetId = const Value.absent(),
    this.assetTag = const Value.absent(),
    this.purpose = const Value.absent(),
    this.state = const Value.absent(),
    this.serverVersion = const Value.absent(),
    this.verifiedPrerequisites = const Value.absent(),
    this.cachedAt = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  CachedExecutionsCompanion.insert({
    required String id,
    required String siteId,
    required String incidentId,
    required String assetId,
    required String assetTag,
    required String purpose,
    required String state,
    required int serverVersion,
    this.verifiedPrerequisites = const Value.absent(),
    required DateTime cachedAt,
    this.rowid = const Value.absent(),
  }) : id = Value(id),
       siteId = Value(siteId),
       incidentId = Value(incidentId),
       assetId = Value(assetId),
       assetTag = Value(assetTag),
       purpose = Value(purpose),
       state = Value(state),
       serverVersion = Value(serverVersion),
       cachedAt = Value(cachedAt);
  static Insertable<CachedExecution> custom({
    Expression<String>? id,
    Expression<String>? siteId,
    Expression<String>? incidentId,
    Expression<String>? assetId,
    Expression<String>? assetTag,
    Expression<String>? purpose,
    Expression<String>? state,
    Expression<int>? serverVersion,
    Expression<String>? verifiedPrerequisites,
    Expression<DateTime>? cachedAt,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (id != null) 'id': id,
      if (siteId != null) 'site_id': siteId,
      if (incidentId != null) 'incident_id': incidentId,
      if (assetId != null) 'asset_id': assetId,
      if (assetTag != null) 'asset_tag': assetTag,
      if (purpose != null) 'purpose': purpose,
      if (state != null) 'state': state,
      if (serverVersion != null) 'server_version': serverVersion,
      if (verifiedPrerequisites != null)
        'verified_prerequisites': verifiedPrerequisites,
      if (cachedAt != null) 'cached_at': cachedAt,
      if (rowid != null) 'rowid': rowid,
    });
  }

  CachedExecutionsCompanion copyWith({
    Value<String>? id,
    Value<String>? siteId,
    Value<String>? incidentId,
    Value<String>? assetId,
    Value<String>? assetTag,
    Value<String>? purpose,
    Value<String>? state,
    Value<int>? serverVersion,
    Value<String>? verifiedPrerequisites,
    Value<DateTime>? cachedAt,
    Value<int>? rowid,
  }) {
    return CachedExecutionsCompanion(
      id: id ?? this.id,
      siteId: siteId ?? this.siteId,
      incidentId: incidentId ?? this.incidentId,
      assetId: assetId ?? this.assetId,
      assetTag: assetTag ?? this.assetTag,
      purpose: purpose ?? this.purpose,
      state: state ?? this.state,
      serverVersion: serverVersion ?? this.serverVersion,
      verifiedPrerequisites:
          verifiedPrerequisites ?? this.verifiedPrerequisites,
      cachedAt: cachedAt ?? this.cachedAt,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (id.present) {
      map['id'] = Variable<String>(id.value);
    }
    if (siteId.present) {
      map['site_id'] = Variable<String>(siteId.value);
    }
    if (incidentId.present) {
      map['incident_id'] = Variable<String>(incidentId.value);
    }
    if (assetId.present) {
      map['asset_id'] = Variable<String>(assetId.value);
    }
    if (assetTag.present) {
      map['asset_tag'] = Variable<String>(assetTag.value);
    }
    if (purpose.present) {
      map['purpose'] = Variable<String>(purpose.value);
    }
    if (state.present) {
      map['state'] = Variable<String>(state.value);
    }
    if (serverVersion.present) {
      map['server_version'] = Variable<int>(serverVersion.value);
    }
    if (verifiedPrerequisites.present) {
      map['verified_prerequisites'] = Variable<String>(
        verifiedPrerequisites.value,
      );
    }
    if (cachedAt.present) {
      map['cached_at'] = Variable<DateTime>(cachedAt.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('CachedExecutionsCompanion(')
          ..write('id: $id, ')
          ..write('siteId: $siteId, ')
          ..write('incidentId: $incidentId, ')
          ..write('assetId: $assetId, ')
          ..write('assetTag: $assetTag, ')
          ..write('purpose: $purpose, ')
          ..write('state: $state, ')
          ..write('serverVersion: $serverVersion, ')
          ..write('verifiedPrerequisites: $verifiedPrerequisites, ')
          ..write('cachedAt: $cachedAt, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $CachedExecutionStepsTable extends CachedExecutionSteps
    with TableInfo<$CachedExecutionStepsTable, CachedExecutionStep> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $CachedExecutionStepsTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _idMeta = const VerificationMeta('id');
  @override
  late final GeneratedColumn<String> id = GeneratedColumn<String>(
    'id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _executionIdMeta = const VerificationMeta(
    'executionId',
  );
  @override
  late final GeneratedColumn<String> executionId = GeneratedColumn<String>(
    'execution_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _stepKeyMeta = const VerificationMeta(
    'stepKey',
  );
  @override
  late final GeneratedColumn<String> stepKey = GeneratedColumn<String>(
    'step_key',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _sequenceMeta = const VerificationMeta(
    'sequence',
  );
  @override
  late final GeneratedColumn<int> sequence = GeneratedColumn<int>(
    'sequence',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _titleMeta = const VerificationMeta('title');
  @override
  late final GeneratedColumn<String> title = GeneratedColumn<String>(
    'title',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _stateMeta = const VerificationMeta('state');
  @override
  late final GeneratedColumn<String> state = GeneratedColumn<String>(
    'state',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _riskLevelMeta = const VerificationMeta(
    'riskLevel',
  );
  @override
  late final GeneratedColumn<String> riskLevel = GeneratedColumn<String>(
    'risk_level',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _requiredPrerequisiteMeta =
      const VerificationMeta('requiredPrerequisite');
  @override
  late final GeneratedColumn<String> requiredPrerequisite =
      GeneratedColumn<String>(
        'required_prerequisite',
        aliasedName,
        true,
        type: DriftSqlType.string,
        requiredDuringInsert: false,
      );
  static const VerificationMeta _blockedReasonMeta = const VerificationMeta(
    'blockedReason',
  );
  @override
  late final GeneratedColumn<String> blockedReason = GeneratedColumn<String>(
    'blocked_reason',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _serverVersionMeta = const VerificationMeta(
    'serverVersion',
  );
  @override
  late final GeneratedColumn<int> serverVersion = GeneratedColumn<int>(
    'server_version',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  @override
  List<GeneratedColumn> get $columns => [
    id,
    executionId,
    stepKey,
    sequence,
    title,
    state,
    riskLevel,
    requiredPrerequisite,
    blockedReason,
    serverVersion,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'cached_execution_steps';
  @override
  VerificationContext validateIntegrity(
    Insertable<CachedExecutionStep> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('id')) {
      context.handle(_idMeta, id.isAcceptableOrUnknown(data['id']!, _idMeta));
    } else if (isInserting) {
      context.missing(_idMeta);
    }
    if (data.containsKey('execution_id')) {
      context.handle(
        _executionIdMeta,
        executionId.isAcceptableOrUnknown(
          data['execution_id']!,
          _executionIdMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_executionIdMeta);
    }
    if (data.containsKey('step_key')) {
      context.handle(
        _stepKeyMeta,
        stepKey.isAcceptableOrUnknown(data['step_key']!, _stepKeyMeta),
      );
    } else if (isInserting) {
      context.missing(_stepKeyMeta);
    }
    if (data.containsKey('sequence')) {
      context.handle(
        _sequenceMeta,
        sequence.isAcceptableOrUnknown(data['sequence']!, _sequenceMeta),
      );
    } else if (isInserting) {
      context.missing(_sequenceMeta);
    }
    if (data.containsKey('title')) {
      context.handle(
        _titleMeta,
        title.isAcceptableOrUnknown(data['title']!, _titleMeta),
      );
    } else if (isInserting) {
      context.missing(_titleMeta);
    }
    if (data.containsKey('state')) {
      context.handle(
        _stateMeta,
        state.isAcceptableOrUnknown(data['state']!, _stateMeta),
      );
    } else if (isInserting) {
      context.missing(_stateMeta);
    }
    if (data.containsKey('risk_level')) {
      context.handle(
        _riskLevelMeta,
        riskLevel.isAcceptableOrUnknown(data['risk_level']!, _riskLevelMeta),
      );
    } else if (isInserting) {
      context.missing(_riskLevelMeta);
    }
    if (data.containsKey('required_prerequisite')) {
      context.handle(
        _requiredPrerequisiteMeta,
        requiredPrerequisite.isAcceptableOrUnknown(
          data['required_prerequisite']!,
          _requiredPrerequisiteMeta,
        ),
      );
    }
    if (data.containsKey('blocked_reason')) {
      context.handle(
        _blockedReasonMeta,
        blockedReason.isAcceptableOrUnknown(
          data['blocked_reason']!,
          _blockedReasonMeta,
        ),
      );
    }
    if (data.containsKey('server_version')) {
      context.handle(
        _serverVersionMeta,
        serverVersion.isAcceptableOrUnknown(
          data['server_version']!,
          _serverVersionMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_serverVersionMeta);
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {id};
  @override
  CachedExecutionStep map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return CachedExecutionStep(
      id: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}id'],
      )!,
      executionId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}execution_id'],
      )!,
      stepKey: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}step_key'],
      )!,
      sequence: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}sequence'],
      )!,
      title: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}title'],
      )!,
      state: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}state'],
      )!,
      riskLevel: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}risk_level'],
      )!,
      requiredPrerequisite: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}required_prerequisite'],
      ),
      blockedReason: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}blocked_reason'],
      ),
      serverVersion: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}server_version'],
      )!,
    );
  }

  @override
  $CachedExecutionStepsTable createAlias(String alias) {
    return $CachedExecutionStepsTable(attachedDatabase, alias);
  }
}

class CachedExecutionStep extends DataClass
    implements Insertable<CachedExecutionStep> {
  final String id;
  final String executionId;
  final String stepKey;
  final int sequence;
  final String title;
  final String state;
  final String riskLevel;
  final String? requiredPrerequisite;
  final String? blockedReason;
  final int serverVersion;
  const CachedExecutionStep({
    required this.id,
    required this.executionId,
    required this.stepKey,
    required this.sequence,
    required this.title,
    required this.state,
    required this.riskLevel,
    this.requiredPrerequisite,
    this.blockedReason,
    required this.serverVersion,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['id'] = Variable<String>(id);
    map['execution_id'] = Variable<String>(executionId);
    map['step_key'] = Variable<String>(stepKey);
    map['sequence'] = Variable<int>(sequence);
    map['title'] = Variable<String>(title);
    map['state'] = Variable<String>(state);
    map['risk_level'] = Variable<String>(riskLevel);
    if (!nullToAbsent || requiredPrerequisite != null) {
      map['required_prerequisite'] = Variable<String>(requiredPrerequisite);
    }
    if (!nullToAbsent || blockedReason != null) {
      map['blocked_reason'] = Variable<String>(blockedReason);
    }
    map['server_version'] = Variable<int>(serverVersion);
    return map;
  }

  CachedExecutionStepsCompanion toCompanion(bool nullToAbsent) {
    return CachedExecutionStepsCompanion(
      id: Value(id),
      executionId: Value(executionId),
      stepKey: Value(stepKey),
      sequence: Value(sequence),
      title: Value(title),
      state: Value(state),
      riskLevel: Value(riskLevel),
      requiredPrerequisite: requiredPrerequisite == null && nullToAbsent
          ? const Value.absent()
          : Value(requiredPrerequisite),
      blockedReason: blockedReason == null && nullToAbsent
          ? const Value.absent()
          : Value(blockedReason),
      serverVersion: Value(serverVersion),
    );
  }

  factory CachedExecutionStep.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return CachedExecutionStep(
      id: serializer.fromJson<String>(json['id']),
      executionId: serializer.fromJson<String>(json['executionId']),
      stepKey: serializer.fromJson<String>(json['stepKey']),
      sequence: serializer.fromJson<int>(json['sequence']),
      title: serializer.fromJson<String>(json['title']),
      state: serializer.fromJson<String>(json['state']),
      riskLevel: serializer.fromJson<String>(json['riskLevel']),
      requiredPrerequisite: serializer.fromJson<String?>(
        json['requiredPrerequisite'],
      ),
      blockedReason: serializer.fromJson<String?>(json['blockedReason']),
      serverVersion: serializer.fromJson<int>(json['serverVersion']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'id': serializer.toJson<String>(id),
      'executionId': serializer.toJson<String>(executionId),
      'stepKey': serializer.toJson<String>(stepKey),
      'sequence': serializer.toJson<int>(sequence),
      'title': serializer.toJson<String>(title),
      'state': serializer.toJson<String>(state),
      'riskLevel': serializer.toJson<String>(riskLevel),
      'requiredPrerequisite': serializer.toJson<String?>(requiredPrerequisite),
      'blockedReason': serializer.toJson<String?>(blockedReason),
      'serverVersion': serializer.toJson<int>(serverVersion),
    };
  }

  CachedExecutionStep copyWith({
    String? id,
    String? executionId,
    String? stepKey,
    int? sequence,
    String? title,
    String? state,
    String? riskLevel,
    Value<String?> requiredPrerequisite = const Value.absent(),
    Value<String?> blockedReason = const Value.absent(),
    int? serverVersion,
  }) => CachedExecutionStep(
    id: id ?? this.id,
    executionId: executionId ?? this.executionId,
    stepKey: stepKey ?? this.stepKey,
    sequence: sequence ?? this.sequence,
    title: title ?? this.title,
    state: state ?? this.state,
    riskLevel: riskLevel ?? this.riskLevel,
    requiredPrerequisite: requiredPrerequisite.present
        ? requiredPrerequisite.value
        : this.requiredPrerequisite,
    blockedReason: blockedReason.present
        ? blockedReason.value
        : this.blockedReason,
    serverVersion: serverVersion ?? this.serverVersion,
  );
  CachedExecutionStep copyWithCompanion(CachedExecutionStepsCompanion data) {
    return CachedExecutionStep(
      id: data.id.present ? data.id.value : this.id,
      executionId: data.executionId.present
          ? data.executionId.value
          : this.executionId,
      stepKey: data.stepKey.present ? data.stepKey.value : this.stepKey,
      sequence: data.sequence.present ? data.sequence.value : this.sequence,
      title: data.title.present ? data.title.value : this.title,
      state: data.state.present ? data.state.value : this.state,
      riskLevel: data.riskLevel.present ? data.riskLevel.value : this.riskLevel,
      requiredPrerequisite: data.requiredPrerequisite.present
          ? data.requiredPrerequisite.value
          : this.requiredPrerequisite,
      blockedReason: data.blockedReason.present
          ? data.blockedReason.value
          : this.blockedReason,
      serverVersion: data.serverVersion.present
          ? data.serverVersion.value
          : this.serverVersion,
    );
  }

  @override
  String toString() {
    return (StringBuffer('CachedExecutionStep(')
          ..write('id: $id, ')
          ..write('executionId: $executionId, ')
          ..write('stepKey: $stepKey, ')
          ..write('sequence: $sequence, ')
          ..write('title: $title, ')
          ..write('state: $state, ')
          ..write('riskLevel: $riskLevel, ')
          ..write('requiredPrerequisite: $requiredPrerequisite, ')
          ..write('blockedReason: $blockedReason, ')
          ..write('serverVersion: $serverVersion')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(
    id,
    executionId,
    stepKey,
    sequence,
    title,
    state,
    riskLevel,
    requiredPrerequisite,
    blockedReason,
    serverVersion,
  );
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is CachedExecutionStep &&
          other.id == this.id &&
          other.executionId == this.executionId &&
          other.stepKey == this.stepKey &&
          other.sequence == this.sequence &&
          other.title == this.title &&
          other.state == this.state &&
          other.riskLevel == this.riskLevel &&
          other.requiredPrerequisite == this.requiredPrerequisite &&
          other.blockedReason == this.blockedReason &&
          other.serverVersion == this.serverVersion);
}

class CachedExecutionStepsCompanion
    extends UpdateCompanion<CachedExecutionStep> {
  final Value<String> id;
  final Value<String> executionId;
  final Value<String> stepKey;
  final Value<int> sequence;
  final Value<String> title;
  final Value<String> state;
  final Value<String> riskLevel;
  final Value<String?> requiredPrerequisite;
  final Value<String?> blockedReason;
  final Value<int> serverVersion;
  final Value<int> rowid;
  const CachedExecutionStepsCompanion({
    this.id = const Value.absent(),
    this.executionId = const Value.absent(),
    this.stepKey = const Value.absent(),
    this.sequence = const Value.absent(),
    this.title = const Value.absent(),
    this.state = const Value.absent(),
    this.riskLevel = const Value.absent(),
    this.requiredPrerequisite = const Value.absent(),
    this.blockedReason = const Value.absent(),
    this.serverVersion = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  CachedExecutionStepsCompanion.insert({
    required String id,
    required String executionId,
    required String stepKey,
    required int sequence,
    required String title,
    required String state,
    required String riskLevel,
    this.requiredPrerequisite = const Value.absent(),
    this.blockedReason = const Value.absent(),
    required int serverVersion,
    this.rowid = const Value.absent(),
  }) : id = Value(id),
       executionId = Value(executionId),
       stepKey = Value(stepKey),
       sequence = Value(sequence),
       title = Value(title),
       state = Value(state),
       riskLevel = Value(riskLevel),
       serverVersion = Value(serverVersion);
  static Insertable<CachedExecutionStep> custom({
    Expression<String>? id,
    Expression<String>? executionId,
    Expression<String>? stepKey,
    Expression<int>? sequence,
    Expression<String>? title,
    Expression<String>? state,
    Expression<String>? riskLevel,
    Expression<String>? requiredPrerequisite,
    Expression<String>? blockedReason,
    Expression<int>? serverVersion,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (id != null) 'id': id,
      if (executionId != null) 'execution_id': executionId,
      if (stepKey != null) 'step_key': stepKey,
      if (sequence != null) 'sequence': sequence,
      if (title != null) 'title': title,
      if (state != null) 'state': state,
      if (riskLevel != null) 'risk_level': riskLevel,
      if (requiredPrerequisite != null)
        'required_prerequisite': requiredPrerequisite,
      if (blockedReason != null) 'blocked_reason': blockedReason,
      if (serverVersion != null) 'server_version': serverVersion,
      if (rowid != null) 'rowid': rowid,
    });
  }

  CachedExecutionStepsCompanion copyWith({
    Value<String>? id,
    Value<String>? executionId,
    Value<String>? stepKey,
    Value<int>? sequence,
    Value<String>? title,
    Value<String>? state,
    Value<String>? riskLevel,
    Value<String?>? requiredPrerequisite,
    Value<String?>? blockedReason,
    Value<int>? serverVersion,
    Value<int>? rowid,
  }) {
    return CachedExecutionStepsCompanion(
      id: id ?? this.id,
      executionId: executionId ?? this.executionId,
      stepKey: stepKey ?? this.stepKey,
      sequence: sequence ?? this.sequence,
      title: title ?? this.title,
      state: state ?? this.state,
      riskLevel: riskLevel ?? this.riskLevel,
      requiredPrerequisite: requiredPrerequisite ?? this.requiredPrerequisite,
      blockedReason: blockedReason ?? this.blockedReason,
      serverVersion: serverVersion ?? this.serverVersion,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (id.present) {
      map['id'] = Variable<String>(id.value);
    }
    if (executionId.present) {
      map['execution_id'] = Variable<String>(executionId.value);
    }
    if (stepKey.present) {
      map['step_key'] = Variable<String>(stepKey.value);
    }
    if (sequence.present) {
      map['sequence'] = Variable<int>(sequence.value);
    }
    if (title.present) {
      map['title'] = Variable<String>(title.value);
    }
    if (state.present) {
      map['state'] = Variable<String>(state.value);
    }
    if (riskLevel.present) {
      map['risk_level'] = Variable<String>(riskLevel.value);
    }
    if (requiredPrerequisite.present) {
      map['required_prerequisite'] = Variable<String>(
        requiredPrerequisite.value,
      );
    }
    if (blockedReason.present) {
      map['blocked_reason'] = Variable<String>(blockedReason.value);
    }
    if (serverVersion.present) {
      map['server_version'] = Variable<int>(serverVersion.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('CachedExecutionStepsCompanion(')
          ..write('id: $id, ')
          ..write('executionId: $executionId, ')
          ..write('stepKey: $stepKey, ')
          ..write('sequence: $sequence, ')
          ..write('title: $title, ')
          ..write('state: $state, ')
          ..write('riskLevel: $riskLevel, ')
          ..write('requiredPrerequisite: $requiredPrerequisite, ')
          ..write('blockedReason: $blockedReason, ')
          ..write('serverVersion: $serverVersion, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $LocalMeasurementsTable extends LocalMeasurements
    with TableInfo<$LocalMeasurementsTable, LocalMeasurement> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $LocalMeasurementsTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _clientEventIdMeta = const VerificationMeta(
    'clientEventId',
  );
  @override
  late final GeneratedColumn<String> clientEventId = GeneratedColumn<String>(
    'client_event_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _executionIdMeta = const VerificationMeta(
    'executionId',
  );
  @override
  late final GeneratedColumn<String> executionId = GeneratedColumn<String>(
    'execution_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _componentIdMeta = const VerificationMeta(
    'componentId',
  );
  @override
  late final GeneratedColumn<String> componentId = GeneratedColumn<String>(
    'component_id',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _measurementTypeMeta = const VerificationMeta(
    'measurementType',
  );
  @override
  late final GeneratedColumn<String> measurementType = GeneratedColumn<String>(
    'measurement_type',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _valueMeta = const VerificationMeta('value');
  @override
  late final GeneratedColumn<String> value = GeneratedColumn<String>(
    'value',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _unitMeta = const VerificationMeta('unit');
  @override
  late final GeneratedColumn<String> unit = GeneratedColumn<String>(
    'unit',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _sourceMeta = const VerificationMeta('source');
  @override
  late final GeneratedColumn<String> source = GeneratedColumn<String>(
    'source',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
    defaultValue: const Constant('MANUAL'),
  );
  static const VerificationMeta _dataQualityMeta = const VerificationMeta(
    'dataQuality',
  );
  @override
  late final GeneratedColumn<String> dataQuality = GeneratedColumn<String>(
    'data_quality',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
    defaultValue: const Constant('GOOD'),
  );
  static const VerificationMeta _verificationStatusMeta =
      const VerificationMeta('verificationStatus');
  @override
  late final GeneratedColumn<String> verificationStatus =
      GeneratedColumn<String>(
        'verification_status',
        aliasedName,
        false,
        type: DriftSqlType.string,
        requiredDuringInsert: false,
        defaultValue: const Constant('UNVERIFIED'),
      );
  static const VerificationMeta _observedAtMeta = const VerificationMeta(
    'observedAt',
  );
  @override
  late final GeneratedColumn<DateTime> observedAt = GeneratedColumn<DateTime>(
    'observed_at',
    aliasedName,
    false,
    type: DriftSqlType.dateTime,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _createdAtDeviceMeta = const VerificationMeta(
    'createdAtDevice',
  );
  @override
  late final GeneratedColumn<DateTime> createdAtDevice =
      GeneratedColumn<DateTime>(
        'created_at_device',
        aliasedName,
        false,
        type: DriftSqlType.dateTime,
        requiredDuringInsert: true,
      );
  static const VerificationMeta _syncStatusMeta = const VerificationMeta(
    'syncStatus',
  );
  @override
  late final GeneratedColumn<String> syncStatus = GeneratedColumn<String>(
    'sync_status',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
    defaultValue: const Constant('PENDING'),
  );
  static const VerificationMeta _serverIdMeta = const VerificationMeta(
    'serverId',
  );
  @override
  late final GeneratedColumn<String> serverId = GeneratedColumn<String>(
    'server_id',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _syncErrorMeta = const VerificationMeta(
    'syncError',
  );
  @override
  late final GeneratedColumn<String> syncError = GeneratedColumn<String>(
    'sync_error',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  @override
  List<GeneratedColumn> get $columns => [
    clientEventId,
    executionId,
    componentId,
    measurementType,
    value,
    unit,
    source,
    dataQuality,
    verificationStatus,
    observedAt,
    createdAtDevice,
    syncStatus,
    serverId,
    syncError,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'local_measurements';
  @override
  VerificationContext validateIntegrity(
    Insertable<LocalMeasurement> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('client_event_id')) {
      context.handle(
        _clientEventIdMeta,
        clientEventId.isAcceptableOrUnknown(
          data['client_event_id']!,
          _clientEventIdMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_clientEventIdMeta);
    }
    if (data.containsKey('execution_id')) {
      context.handle(
        _executionIdMeta,
        executionId.isAcceptableOrUnknown(
          data['execution_id']!,
          _executionIdMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_executionIdMeta);
    }
    if (data.containsKey('component_id')) {
      context.handle(
        _componentIdMeta,
        componentId.isAcceptableOrUnknown(
          data['component_id']!,
          _componentIdMeta,
        ),
      );
    }
    if (data.containsKey('measurement_type')) {
      context.handle(
        _measurementTypeMeta,
        measurementType.isAcceptableOrUnknown(
          data['measurement_type']!,
          _measurementTypeMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_measurementTypeMeta);
    }
    if (data.containsKey('value')) {
      context.handle(
        _valueMeta,
        value.isAcceptableOrUnknown(data['value']!, _valueMeta),
      );
    } else if (isInserting) {
      context.missing(_valueMeta);
    }
    if (data.containsKey('unit')) {
      context.handle(
        _unitMeta,
        unit.isAcceptableOrUnknown(data['unit']!, _unitMeta),
      );
    } else if (isInserting) {
      context.missing(_unitMeta);
    }
    if (data.containsKey('source')) {
      context.handle(
        _sourceMeta,
        source.isAcceptableOrUnknown(data['source']!, _sourceMeta),
      );
    }
    if (data.containsKey('data_quality')) {
      context.handle(
        _dataQualityMeta,
        dataQuality.isAcceptableOrUnknown(
          data['data_quality']!,
          _dataQualityMeta,
        ),
      );
    }
    if (data.containsKey('verification_status')) {
      context.handle(
        _verificationStatusMeta,
        verificationStatus.isAcceptableOrUnknown(
          data['verification_status']!,
          _verificationStatusMeta,
        ),
      );
    }
    if (data.containsKey('observed_at')) {
      context.handle(
        _observedAtMeta,
        observedAt.isAcceptableOrUnknown(data['observed_at']!, _observedAtMeta),
      );
    } else if (isInserting) {
      context.missing(_observedAtMeta);
    }
    if (data.containsKey('created_at_device')) {
      context.handle(
        _createdAtDeviceMeta,
        createdAtDevice.isAcceptableOrUnknown(
          data['created_at_device']!,
          _createdAtDeviceMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_createdAtDeviceMeta);
    }
    if (data.containsKey('sync_status')) {
      context.handle(
        _syncStatusMeta,
        syncStatus.isAcceptableOrUnknown(data['sync_status']!, _syncStatusMeta),
      );
    }
    if (data.containsKey('server_id')) {
      context.handle(
        _serverIdMeta,
        serverId.isAcceptableOrUnknown(data['server_id']!, _serverIdMeta),
      );
    }
    if (data.containsKey('sync_error')) {
      context.handle(
        _syncErrorMeta,
        syncError.isAcceptableOrUnknown(data['sync_error']!, _syncErrorMeta),
      );
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {clientEventId};
  @override
  LocalMeasurement map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return LocalMeasurement(
      clientEventId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}client_event_id'],
      )!,
      executionId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}execution_id'],
      )!,
      componentId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}component_id'],
      ),
      measurementType: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}measurement_type'],
      )!,
      value: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}value'],
      )!,
      unit: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}unit'],
      )!,
      source: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}source'],
      )!,
      dataQuality: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}data_quality'],
      )!,
      verificationStatus: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}verification_status'],
      )!,
      observedAt: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}observed_at'],
      )!,
      createdAtDevice: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}created_at_device'],
      )!,
      syncStatus: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}sync_status'],
      )!,
      serverId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}server_id'],
      ),
      syncError: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}sync_error'],
      ),
    );
  }

  @override
  $LocalMeasurementsTable createAlias(String alias) {
    return $LocalMeasurementsTable(attachedDatabase, alias);
  }
}

class LocalMeasurement extends DataClass
    implements Insertable<LocalMeasurement> {
  final String clientEventId;
  final String executionId;
  final String? componentId;
  final String measurementType;
  final String value;
  final String unit;
  final String source;
  final String dataQuality;
  final String verificationStatus;
  final DateTime observedAt;
  final DateTime createdAtDevice;
  final String syncStatus;
  final String? serverId;
  final String? syncError;
  const LocalMeasurement({
    required this.clientEventId,
    required this.executionId,
    this.componentId,
    required this.measurementType,
    required this.value,
    required this.unit,
    required this.source,
    required this.dataQuality,
    required this.verificationStatus,
    required this.observedAt,
    required this.createdAtDevice,
    required this.syncStatus,
    this.serverId,
    this.syncError,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['client_event_id'] = Variable<String>(clientEventId);
    map['execution_id'] = Variable<String>(executionId);
    if (!nullToAbsent || componentId != null) {
      map['component_id'] = Variable<String>(componentId);
    }
    map['measurement_type'] = Variable<String>(measurementType);
    map['value'] = Variable<String>(value);
    map['unit'] = Variable<String>(unit);
    map['source'] = Variable<String>(source);
    map['data_quality'] = Variable<String>(dataQuality);
    map['verification_status'] = Variable<String>(verificationStatus);
    map['observed_at'] = Variable<DateTime>(observedAt);
    map['created_at_device'] = Variable<DateTime>(createdAtDevice);
    map['sync_status'] = Variable<String>(syncStatus);
    if (!nullToAbsent || serverId != null) {
      map['server_id'] = Variable<String>(serverId);
    }
    if (!nullToAbsent || syncError != null) {
      map['sync_error'] = Variable<String>(syncError);
    }
    return map;
  }

  LocalMeasurementsCompanion toCompanion(bool nullToAbsent) {
    return LocalMeasurementsCompanion(
      clientEventId: Value(clientEventId),
      executionId: Value(executionId),
      componentId: componentId == null && nullToAbsent
          ? const Value.absent()
          : Value(componentId),
      measurementType: Value(measurementType),
      value: Value(value),
      unit: Value(unit),
      source: Value(source),
      dataQuality: Value(dataQuality),
      verificationStatus: Value(verificationStatus),
      observedAt: Value(observedAt),
      createdAtDevice: Value(createdAtDevice),
      syncStatus: Value(syncStatus),
      serverId: serverId == null && nullToAbsent
          ? const Value.absent()
          : Value(serverId),
      syncError: syncError == null && nullToAbsent
          ? const Value.absent()
          : Value(syncError),
    );
  }

  factory LocalMeasurement.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return LocalMeasurement(
      clientEventId: serializer.fromJson<String>(json['clientEventId']),
      executionId: serializer.fromJson<String>(json['executionId']),
      componentId: serializer.fromJson<String?>(json['componentId']),
      measurementType: serializer.fromJson<String>(json['measurementType']),
      value: serializer.fromJson<String>(json['value']),
      unit: serializer.fromJson<String>(json['unit']),
      source: serializer.fromJson<String>(json['source']),
      dataQuality: serializer.fromJson<String>(json['dataQuality']),
      verificationStatus: serializer.fromJson<String>(
        json['verificationStatus'],
      ),
      observedAt: serializer.fromJson<DateTime>(json['observedAt']),
      createdAtDevice: serializer.fromJson<DateTime>(json['createdAtDevice']),
      syncStatus: serializer.fromJson<String>(json['syncStatus']),
      serverId: serializer.fromJson<String?>(json['serverId']),
      syncError: serializer.fromJson<String?>(json['syncError']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'clientEventId': serializer.toJson<String>(clientEventId),
      'executionId': serializer.toJson<String>(executionId),
      'componentId': serializer.toJson<String?>(componentId),
      'measurementType': serializer.toJson<String>(measurementType),
      'value': serializer.toJson<String>(value),
      'unit': serializer.toJson<String>(unit),
      'source': serializer.toJson<String>(source),
      'dataQuality': serializer.toJson<String>(dataQuality),
      'verificationStatus': serializer.toJson<String>(verificationStatus),
      'observedAt': serializer.toJson<DateTime>(observedAt),
      'createdAtDevice': serializer.toJson<DateTime>(createdAtDevice),
      'syncStatus': serializer.toJson<String>(syncStatus),
      'serverId': serializer.toJson<String?>(serverId),
      'syncError': serializer.toJson<String?>(syncError),
    };
  }

  LocalMeasurement copyWith({
    String? clientEventId,
    String? executionId,
    Value<String?> componentId = const Value.absent(),
    String? measurementType,
    String? value,
    String? unit,
    String? source,
    String? dataQuality,
    String? verificationStatus,
    DateTime? observedAt,
    DateTime? createdAtDevice,
    String? syncStatus,
    Value<String?> serverId = const Value.absent(),
    Value<String?> syncError = const Value.absent(),
  }) => LocalMeasurement(
    clientEventId: clientEventId ?? this.clientEventId,
    executionId: executionId ?? this.executionId,
    componentId: componentId.present ? componentId.value : this.componentId,
    measurementType: measurementType ?? this.measurementType,
    value: value ?? this.value,
    unit: unit ?? this.unit,
    source: source ?? this.source,
    dataQuality: dataQuality ?? this.dataQuality,
    verificationStatus: verificationStatus ?? this.verificationStatus,
    observedAt: observedAt ?? this.observedAt,
    createdAtDevice: createdAtDevice ?? this.createdAtDevice,
    syncStatus: syncStatus ?? this.syncStatus,
    serverId: serverId.present ? serverId.value : this.serverId,
    syncError: syncError.present ? syncError.value : this.syncError,
  );
  LocalMeasurement copyWithCompanion(LocalMeasurementsCompanion data) {
    return LocalMeasurement(
      clientEventId: data.clientEventId.present
          ? data.clientEventId.value
          : this.clientEventId,
      executionId: data.executionId.present
          ? data.executionId.value
          : this.executionId,
      componentId: data.componentId.present
          ? data.componentId.value
          : this.componentId,
      measurementType: data.measurementType.present
          ? data.measurementType.value
          : this.measurementType,
      value: data.value.present ? data.value.value : this.value,
      unit: data.unit.present ? data.unit.value : this.unit,
      source: data.source.present ? data.source.value : this.source,
      dataQuality: data.dataQuality.present
          ? data.dataQuality.value
          : this.dataQuality,
      verificationStatus: data.verificationStatus.present
          ? data.verificationStatus.value
          : this.verificationStatus,
      observedAt: data.observedAt.present
          ? data.observedAt.value
          : this.observedAt,
      createdAtDevice: data.createdAtDevice.present
          ? data.createdAtDevice.value
          : this.createdAtDevice,
      syncStatus: data.syncStatus.present
          ? data.syncStatus.value
          : this.syncStatus,
      serverId: data.serverId.present ? data.serverId.value : this.serverId,
      syncError: data.syncError.present ? data.syncError.value : this.syncError,
    );
  }

  @override
  String toString() {
    return (StringBuffer('LocalMeasurement(')
          ..write('clientEventId: $clientEventId, ')
          ..write('executionId: $executionId, ')
          ..write('componentId: $componentId, ')
          ..write('measurementType: $measurementType, ')
          ..write('value: $value, ')
          ..write('unit: $unit, ')
          ..write('source: $source, ')
          ..write('dataQuality: $dataQuality, ')
          ..write('verificationStatus: $verificationStatus, ')
          ..write('observedAt: $observedAt, ')
          ..write('createdAtDevice: $createdAtDevice, ')
          ..write('syncStatus: $syncStatus, ')
          ..write('serverId: $serverId, ')
          ..write('syncError: $syncError')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(
    clientEventId,
    executionId,
    componentId,
    measurementType,
    value,
    unit,
    source,
    dataQuality,
    verificationStatus,
    observedAt,
    createdAtDevice,
    syncStatus,
    serverId,
    syncError,
  );
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is LocalMeasurement &&
          other.clientEventId == this.clientEventId &&
          other.executionId == this.executionId &&
          other.componentId == this.componentId &&
          other.measurementType == this.measurementType &&
          other.value == this.value &&
          other.unit == this.unit &&
          other.source == this.source &&
          other.dataQuality == this.dataQuality &&
          other.verificationStatus == this.verificationStatus &&
          other.observedAt == this.observedAt &&
          other.createdAtDevice == this.createdAtDevice &&
          other.syncStatus == this.syncStatus &&
          other.serverId == this.serverId &&
          other.syncError == this.syncError);
}

class LocalMeasurementsCompanion extends UpdateCompanion<LocalMeasurement> {
  final Value<String> clientEventId;
  final Value<String> executionId;
  final Value<String?> componentId;
  final Value<String> measurementType;
  final Value<String> value;
  final Value<String> unit;
  final Value<String> source;
  final Value<String> dataQuality;
  final Value<String> verificationStatus;
  final Value<DateTime> observedAt;
  final Value<DateTime> createdAtDevice;
  final Value<String> syncStatus;
  final Value<String?> serverId;
  final Value<String?> syncError;
  final Value<int> rowid;
  const LocalMeasurementsCompanion({
    this.clientEventId = const Value.absent(),
    this.executionId = const Value.absent(),
    this.componentId = const Value.absent(),
    this.measurementType = const Value.absent(),
    this.value = const Value.absent(),
    this.unit = const Value.absent(),
    this.source = const Value.absent(),
    this.dataQuality = const Value.absent(),
    this.verificationStatus = const Value.absent(),
    this.observedAt = const Value.absent(),
    this.createdAtDevice = const Value.absent(),
    this.syncStatus = const Value.absent(),
    this.serverId = const Value.absent(),
    this.syncError = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  LocalMeasurementsCompanion.insert({
    required String clientEventId,
    required String executionId,
    this.componentId = const Value.absent(),
    required String measurementType,
    required String value,
    required String unit,
    this.source = const Value.absent(),
    this.dataQuality = const Value.absent(),
    this.verificationStatus = const Value.absent(),
    required DateTime observedAt,
    required DateTime createdAtDevice,
    this.syncStatus = const Value.absent(),
    this.serverId = const Value.absent(),
    this.syncError = const Value.absent(),
    this.rowid = const Value.absent(),
  }) : clientEventId = Value(clientEventId),
       executionId = Value(executionId),
       measurementType = Value(measurementType),
       value = Value(value),
       unit = Value(unit),
       observedAt = Value(observedAt),
       createdAtDevice = Value(createdAtDevice);
  static Insertable<LocalMeasurement> custom({
    Expression<String>? clientEventId,
    Expression<String>? executionId,
    Expression<String>? componentId,
    Expression<String>? measurementType,
    Expression<String>? value,
    Expression<String>? unit,
    Expression<String>? source,
    Expression<String>? dataQuality,
    Expression<String>? verificationStatus,
    Expression<DateTime>? observedAt,
    Expression<DateTime>? createdAtDevice,
    Expression<String>? syncStatus,
    Expression<String>? serverId,
    Expression<String>? syncError,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (clientEventId != null) 'client_event_id': clientEventId,
      if (executionId != null) 'execution_id': executionId,
      if (componentId != null) 'component_id': componentId,
      if (measurementType != null) 'measurement_type': measurementType,
      if (value != null) 'value': value,
      if (unit != null) 'unit': unit,
      if (source != null) 'source': source,
      if (dataQuality != null) 'data_quality': dataQuality,
      if (verificationStatus != null) 'verification_status': verificationStatus,
      if (observedAt != null) 'observed_at': observedAt,
      if (createdAtDevice != null) 'created_at_device': createdAtDevice,
      if (syncStatus != null) 'sync_status': syncStatus,
      if (serverId != null) 'server_id': serverId,
      if (syncError != null) 'sync_error': syncError,
      if (rowid != null) 'rowid': rowid,
    });
  }

  LocalMeasurementsCompanion copyWith({
    Value<String>? clientEventId,
    Value<String>? executionId,
    Value<String?>? componentId,
    Value<String>? measurementType,
    Value<String>? value,
    Value<String>? unit,
    Value<String>? source,
    Value<String>? dataQuality,
    Value<String>? verificationStatus,
    Value<DateTime>? observedAt,
    Value<DateTime>? createdAtDevice,
    Value<String>? syncStatus,
    Value<String?>? serverId,
    Value<String?>? syncError,
    Value<int>? rowid,
  }) {
    return LocalMeasurementsCompanion(
      clientEventId: clientEventId ?? this.clientEventId,
      executionId: executionId ?? this.executionId,
      componentId: componentId ?? this.componentId,
      measurementType: measurementType ?? this.measurementType,
      value: value ?? this.value,
      unit: unit ?? this.unit,
      source: source ?? this.source,
      dataQuality: dataQuality ?? this.dataQuality,
      verificationStatus: verificationStatus ?? this.verificationStatus,
      observedAt: observedAt ?? this.observedAt,
      createdAtDevice: createdAtDevice ?? this.createdAtDevice,
      syncStatus: syncStatus ?? this.syncStatus,
      serverId: serverId ?? this.serverId,
      syncError: syncError ?? this.syncError,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (clientEventId.present) {
      map['client_event_id'] = Variable<String>(clientEventId.value);
    }
    if (executionId.present) {
      map['execution_id'] = Variable<String>(executionId.value);
    }
    if (componentId.present) {
      map['component_id'] = Variable<String>(componentId.value);
    }
    if (measurementType.present) {
      map['measurement_type'] = Variable<String>(measurementType.value);
    }
    if (value.present) {
      map['value'] = Variable<String>(value.value);
    }
    if (unit.present) {
      map['unit'] = Variable<String>(unit.value);
    }
    if (source.present) {
      map['source'] = Variable<String>(source.value);
    }
    if (dataQuality.present) {
      map['data_quality'] = Variable<String>(dataQuality.value);
    }
    if (verificationStatus.present) {
      map['verification_status'] = Variable<String>(verificationStatus.value);
    }
    if (observedAt.present) {
      map['observed_at'] = Variable<DateTime>(observedAt.value);
    }
    if (createdAtDevice.present) {
      map['created_at_device'] = Variable<DateTime>(createdAtDevice.value);
    }
    if (syncStatus.present) {
      map['sync_status'] = Variable<String>(syncStatus.value);
    }
    if (serverId.present) {
      map['server_id'] = Variable<String>(serverId.value);
    }
    if (syncError.present) {
      map['sync_error'] = Variable<String>(syncError.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('LocalMeasurementsCompanion(')
          ..write('clientEventId: $clientEventId, ')
          ..write('executionId: $executionId, ')
          ..write('componentId: $componentId, ')
          ..write('measurementType: $measurementType, ')
          ..write('value: $value, ')
          ..write('unit: $unit, ')
          ..write('source: $source, ')
          ..write('dataQuality: $dataQuality, ')
          ..write('verificationStatus: $verificationStatus, ')
          ..write('observedAt: $observedAt, ')
          ..write('createdAtDevice: $createdAtDevice, ')
          ..write('syncStatus: $syncStatus, ')
          ..write('serverId: $serverId, ')
          ..write('syncError: $syncError, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $LocalObservationsTable extends LocalObservations
    with TableInfo<$LocalObservationsTable, LocalObservation> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $LocalObservationsTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _clientEventIdMeta = const VerificationMeta(
    'clientEventId',
  );
  @override
  late final GeneratedColumn<String> clientEventId = GeneratedColumn<String>(
    'client_event_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _executionIdMeta = const VerificationMeta(
    'executionId',
  );
  @override
  late final GeneratedColumn<String> executionId = GeneratedColumn<String>(
    'execution_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _componentIdMeta = const VerificationMeta(
    'componentId',
  );
  @override
  late final GeneratedColumn<String> componentId = GeneratedColumn<String>(
    'component_id',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _propertyMeta = const VerificationMeta(
    'property',
  );
  @override
  late final GeneratedColumn<String> property = GeneratedColumn<String>(
    'property',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _statusMeta = const VerificationMeta('status');
  @override
  late final GeneratedColumn<String> status = GeneratedColumn<String>(
    'status',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _narrativeMeta = const VerificationMeta(
    'narrative',
  );
  @override
  late final GeneratedColumn<String> narrative = GeneratedColumn<String>(
    'narrative',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _sourceMeta = const VerificationMeta('source');
  @override
  late final GeneratedColumn<String> source = GeneratedColumn<String>(
    'source',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
    defaultValue: const Constant('TECHNICIAN'),
  );
  static const VerificationMeta _verificationStatusMeta =
      const VerificationMeta('verificationStatus');
  @override
  late final GeneratedColumn<String> verificationStatus =
      GeneratedColumn<String>(
        'verification_status',
        aliasedName,
        false,
        type: DriftSqlType.string,
        requiredDuringInsert: false,
        defaultValue: const Constant('UNVERIFIED'),
      );
  static const VerificationMeta _observedAtMeta = const VerificationMeta(
    'observedAt',
  );
  @override
  late final GeneratedColumn<DateTime> observedAt = GeneratedColumn<DateTime>(
    'observed_at',
    aliasedName,
    false,
    type: DriftSqlType.dateTime,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _createdAtDeviceMeta = const VerificationMeta(
    'createdAtDevice',
  );
  @override
  late final GeneratedColumn<DateTime> createdAtDevice =
      GeneratedColumn<DateTime>(
        'created_at_device',
        aliasedName,
        false,
        type: DriftSqlType.dateTime,
        requiredDuringInsert: true,
      );
  static const VerificationMeta _syncStatusMeta = const VerificationMeta(
    'syncStatus',
  );
  @override
  late final GeneratedColumn<String> syncStatus = GeneratedColumn<String>(
    'sync_status',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
    defaultValue: const Constant('PENDING'),
  );
  static const VerificationMeta _serverIdMeta = const VerificationMeta(
    'serverId',
  );
  @override
  late final GeneratedColumn<String> serverId = GeneratedColumn<String>(
    'server_id',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _syncErrorMeta = const VerificationMeta(
    'syncError',
  );
  @override
  late final GeneratedColumn<String> syncError = GeneratedColumn<String>(
    'sync_error',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  @override
  List<GeneratedColumn> get $columns => [
    clientEventId,
    executionId,
    componentId,
    property,
    status,
    narrative,
    source,
    verificationStatus,
    observedAt,
    createdAtDevice,
    syncStatus,
    serverId,
    syncError,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'local_observations';
  @override
  VerificationContext validateIntegrity(
    Insertable<LocalObservation> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('client_event_id')) {
      context.handle(
        _clientEventIdMeta,
        clientEventId.isAcceptableOrUnknown(
          data['client_event_id']!,
          _clientEventIdMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_clientEventIdMeta);
    }
    if (data.containsKey('execution_id')) {
      context.handle(
        _executionIdMeta,
        executionId.isAcceptableOrUnknown(
          data['execution_id']!,
          _executionIdMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_executionIdMeta);
    }
    if (data.containsKey('component_id')) {
      context.handle(
        _componentIdMeta,
        componentId.isAcceptableOrUnknown(
          data['component_id']!,
          _componentIdMeta,
        ),
      );
    }
    if (data.containsKey('property')) {
      context.handle(
        _propertyMeta,
        property.isAcceptableOrUnknown(data['property']!, _propertyMeta),
      );
    }
    if (data.containsKey('status')) {
      context.handle(
        _statusMeta,
        status.isAcceptableOrUnknown(data['status']!, _statusMeta),
      );
    }
    if (data.containsKey('narrative')) {
      context.handle(
        _narrativeMeta,
        narrative.isAcceptableOrUnknown(data['narrative']!, _narrativeMeta),
      );
    } else if (isInserting) {
      context.missing(_narrativeMeta);
    }
    if (data.containsKey('source')) {
      context.handle(
        _sourceMeta,
        source.isAcceptableOrUnknown(data['source']!, _sourceMeta),
      );
    }
    if (data.containsKey('verification_status')) {
      context.handle(
        _verificationStatusMeta,
        verificationStatus.isAcceptableOrUnknown(
          data['verification_status']!,
          _verificationStatusMeta,
        ),
      );
    }
    if (data.containsKey('observed_at')) {
      context.handle(
        _observedAtMeta,
        observedAt.isAcceptableOrUnknown(data['observed_at']!, _observedAtMeta),
      );
    } else if (isInserting) {
      context.missing(_observedAtMeta);
    }
    if (data.containsKey('created_at_device')) {
      context.handle(
        _createdAtDeviceMeta,
        createdAtDevice.isAcceptableOrUnknown(
          data['created_at_device']!,
          _createdAtDeviceMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_createdAtDeviceMeta);
    }
    if (data.containsKey('sync_status')) {
      context.handle(
        _syncStatusMeta,
        syncStatus.isAcceptableOrUnknown(data['sync_status']!, _syncStatusMeta),
      );
    }
    if (data.containsKey('server_id')) {
      context.handle(
        _serverIdMeta,
        serverId.isAcceptableOrUnknown(data['server_id']!, _serverIdMeta),
      );
    }
    if (data.containsKey('sync_error')) {
      context.handle(
        _syncErrorMeta,
        syncError.isAcceptableOrUnknown(data['sync_error']!, _syncErrorMeta),
      );
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {clientEventId};
  @override
  LocalObservation map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return LocalObservation(
      clientEventId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}client_event_id'],
      )!,
      executionId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}execution_id'],
      )!,
      componentId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}component_id'],
      ),
      property: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}property'],
      ),
      status: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}status'],
      ),
      narrative: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}narrative'],
      )!,
      source: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}source'],
      )!,
      verificationStatus: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}verification_status'],
      )!,
      observedAt: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}observed_at'],
      )!,
      createdAtDevice: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}created_at_device'],
      )!,
      syncStatus: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}sync_status'],
      )!,
      serverId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}server_id'],
      ),
      syncError: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}sync_error'],
      ),
    );
  }

  @override
  $LocalObservationsTable createAlias(String alias) {
    return $LocalObservationsTable(attachedDatabase, alias);
  }
}

class LocalObservation extends DataClass
    implements Insertable<LocalObservation> {
  final String clientEventId;
  final String executionId;
  final String? componentId;
  final String? property;
  final String? status;
  final String narrative;
  final String source;
  final String verificationStatus;
  final DateTime observedAt;
  final DateTime createdAtDevice;
  final String syncStatus;
  final String? serverId;
  final String? syncError;
  const LocalObservation({
    required this.clientEventId,
    required this.executionId,
    this.componentId,
    this.property,
    this.status,
    required this.narrative,
    required this.source,
    required this.verificationStatus,
    required this.observedAt,
    required this.createdAtDevice,
    required this.syncStatus,
    this.serverId,
    this.syncError,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['client_event_id'] = Variable<String>(clientEventId);
    map['execution_id'] = Variable<String>(executionId);
    if (!nullToAbsent || componentId != null) {
      map['component_id'] = Variable<String>(componentId);
    }
    if (!nullToAbsent || property != null) {
      map['property'] = Variable<String>(property);
    }
    if (!nullToAbsent || status != null) {
      map['status'] = Variable<String>(status);
    }
    map['narrative'] = Variable<String>(narrative);
    map['source'] = Variable<String>(source);
    map['verification_status'] = Variable<String>(verificationStatus);
    map['observed_at'] = Variable<DateTime>(observedAt);
    map['created_at_device'] = Variable<DateTime>(createdAtDevice);
    map['sync_status'] = Variable<String>(syncStatus);
    if (!nullToAbsent || serverId != null) {
      map['server_id'] = Variable<String>(serverId);
    }
    if (!nullToAbsent || syncError != null) {
      map['sync_error'] = Variable<String>(syncError);
    }
    return map;
  }

  LocalObservationsCompanion toCompanion(bool nullToAbsent) {
    return LocalObservationsCompanion(
      clientEventId: Value(clientEventId),
      executionId: Value(executionId),
      componentId: componentId == null && nullToAbsent
          ? const Value.absent()
          : Value(componentId),
      property: property == null && nullToAbsent
          ? const Value.absent()
          : Value(property),
      status: status == null && nullToAbsent
          ? const Value.absent()
          : Value(status),
      narrative: Value(narrative),
      source: Value(source),
      verificationStatus: Value(verificationStatus),
      observedAt: Value(observedAt),
      createdAtDevice: Value(createdAtDevice),
      syncStatus: Value(syncStatus),
      serverId: serverId == null && nullToAbsent
          ? const Value.absent()
          : Value(serverId),
      syncError: syncError == null && nullToAbsent
          ? const Value.absent()
          : Value(syncError),
    );
  }

  factory LocalObservation.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return LocalObservation(
      clientEventId: serializer.fromJson<String>(json['clientEventId']),
      executionId: serializer.fromJson<String>(json['executionId']),
      componentId: serializer.fromJson<String?>(json['componentId']),
      property: serializer.fromJson<String?>(json['property']),
      status: serializer.fromJson<String?>(json['status']),
      narrative: serializer.fromJson<String>(json['narrative']),
      source: serializer.fromJson<String>(json['source']),
      verificationStatus: serializer.fromJson<String>(
        json['verificationStatus'],
      ),
      observedAt: serializer.fromJson<DateTime>(json['observedAt']),
      createdAtDevice: serializer.fromJson<DateTime>(json['createdAtDevice']),
      syncStatus: serializer.fromJson<String>(json['syncStatus']),
      serverId: serializer.fromJson<String?>(json['serverId']),
      syncError: serializer.fromJson<String?>(json['syncError']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'clientEventId': serializer.toJson<String>(clientEventId),
      'executionId': serializer.toJson<String>(executionId),
      'componentId': serializer.toJson<String?>(componentId),
      'property': serializer.toJson<String?>(property),
      'status': serializer.toJson<String?>(status),
      'narrative': serializer.toJson<String>(narrative),
      'source': serializer.toJson<String>(source),
      'verificationStatus': serializer.toJson<String>(verificationStatus),
      'observedAt': serializer.toJson<DateTime>(observedAt),
      'createdAtDevice': serializer.toJson<DateTime>(createdAtDevice),
      'syncStatus': serializer.toJson<String>(syncStatus),
      'serverId': serializer.toJson<String?>(serverId),
      'syncError': serializer.toJson<String?>(syncError),
    };
  }

  LocalObservation copyWith({
    String? clientEventId,
    String? executionId,
    Value<String?> componentId = const Value.absent(),
    Value<String?> property = const Value.absent(),
    Value<String?> status = const Value.absent(),
    String? narrative,
    String? source,
    String? verificationStatus,
    DateTime? observedAt,
    DateTime? createdAtDevice,
    String? syncStatus,
    Value<String?> serverId = const Value.absent(),
    Value<String?> syncError = const Value.absent(),
  }) => LocalObservation(
    clientEventId: clientEventId ?? this.clientEventId,
    executionId: executionId ?? this.executionId,
    componentId: componentId.present ? componentId.value : this.componentId,
    property: property.present ? property.value : this.property,
    status: status.present ? status.value : this.status,
    narrative: narrative ?? this.narrative,
    source: source ?? this.source,
    verificationStatus: verificationStatus ?? this.verificationStatus,
    observedAt: observedAt ?? this.observedAt,
    createdAtDevice: createdAtDevice ?? this.createdAtDevice,
    syncStatus: syncStatus ?? this.syncStatus,
    serverId: serverId.present ? serverId.value : this.serverId,
    syncError: syncError.present ? syncError.value : this.syncError,
  );
  LocalObservation copyWithCompanion(LocalObservationsCompanion data) {
    return LocalObservation(
      clientEventId: data.clientEventId.present
          ? data.clientEventId.value
          : this.clientEventId,
      executionId: data.executionId.present
          ? data.executionId.value
          : this.executionId,
      componentId: data.componentId.present
          ? data.componentId.value
          : this.componentId,
      property: data.property.present ? data.property.value : this.property,
      status: data.status.present ? data.status.value : this.status,
      narrative: data.narrative.present ? data.narrative.value : this.narrative,
      source: data.source.present ? data.source.value : this.source,
      verificationStatus: data.verificationStatus.present
          ? data.verificationStatus.value
          : this.verificationStatus,
      observedAt: data.observedAt.present
          ? data.observedAt.value
          : this.observedAt,
      createdAtDevice: data.createdAtDevice.present
          ? data.createdAtDevice.value
          : this.createdAtDevice,
      syncStatus: data.syncStatus.present
          ? data.syncStatus.value
          : this.syncStatus,
      serverId: data.serverId.present ? data.serverId.value : this.serverId,
      syncError: data.syncError.present ? data.syncError.value : this.syncError,
    );
  }

  @override
  String toString() {
    return (StringBuffer('LocalObservation(')
          ..write('clientEventId: $clientEventId, ')
          ..write('executionId: $executionId, ')
          ..write('componentId: $componentId, ')
          ..write('property: $property, ')
          ..write('status: $status, ')
          ..write('narrative: $narrative, ')
          ..write('source: $source, ')
          ..write('verificationStatus: $verificationStatus, ')
          ..write('observedAt: $observedAt, ')
          ..write('createdAtDevice: $createdAtDevice, ')
          ..write('syncStatus: $syncStatus, ')
          ..write('serverId: $serverId, ')
          ..write('syncError: $syncError')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(
    clientEventId,
    executionId,
    componentId,
    property,
    status,
    narrative,
    source,
    verificationStatus,
    observedAt,
    createdAtDevice,
    syncStatus,
    serverId,
    syncError,
  );
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is LocalObservation &&
          other.clientEventId == this.clientEventId &&
          other.executionId == this.executionId &&
          other.componentId == this.componentId &&
          other.property == this.property &&
          other.status == this.status &&
          other.narrative == this.narrative &&
          other.source == this.source &&
          other.verificationStatus == this.verificationStatus &&
          other.observedAt == this.observedAt &&
          other.createdAtDevice == this.createdAtDevice &&
          other.syncStatus == this.syncStatus &&
          other.serverId == this.serverId &&
          other.syncError == this.syncError);
}

class LocalObservationsCompanion extends UpdateCompanion<LocalObservation> {
  final Value<String> clientEventId;
  final Value<String> executionId;
  final Value<String?> componentId;
  final Value<String?> property;
  final Value<String?> status;
  final Value<String> narrative;
  final Value<String> source;
  final Value<String> verificationStatus;
  final Value<DateTime> observedAt;
  final Value<DateTime> createdAtDevice;
  final Value<String> syncStatus;
  final Value<String?> serverId;
  final Value<String?> syncError;
  final Value<int> rowid;
  const LocalObservationsCompanion({
    this.clientEventId = const Value.absent(),
    this.executionId = const Value.absent(),
    this.componentId = const Value.absent(),
    this.property = const Value.absent(),
    this.status = const Value.absent(),
    this.narrative = const Value.absent(),
    this.source = const Value.absent(),
    this.verificationStatus = const Value.absent(),
    this.observedAt = const Value.absent(),
    this.createdAtDevice = const Value.absent(),
    this.syncStatus = const Value.absent(),
    this.serverId = const Value.absent(),
    this.syncError = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  LocalObservationsCompanion.insert({
    required String clientEventId,
    required String executionId,
    this.componentId = const Value.absent(),
    this.property = const Value.absent(),
    this.status = const Value.absent(),
    required String narrative,
    this.source = const Value.absent(),
    this.verificationStatus = const Value.absent(),
    required DateTime observedAt,
    required DateTime createdAtDevice,
    this.syncStatus = const Value.absent(),
    this.serverId = const Value.absent(),
    this.syncError = const Value.absent(),
    this.rowid = const Value.absent(),
  }) : clientEventId = Value(clientEventId),
       executionId = Value(executionId),
       narrative = Value(narrative),
       observedAt = Value(observedAt),
       createdAtDevice = Value(createdAtDevice);
  static Insertable<LocalObservation> custom({
    Expression<String>? clientEventId,
    Expression<String>? executionId,
    Expression<String>? componentId,
    Expression<String>? property,
    Expression<String>? status,
    Expression<String>? narrative,
    Expression<String>? source,
    Expression<String>? verificationStatus,
    Expression<DateTime>? observedAt,
    Expression<DateTime>? createdAtDevice,
    Expression<String>? syncStatus,
    Expression<String>? serverId,
    Expression<String>? syncError,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (clientEventId != null) 'client_event_id': clientEventId,
      if (executionId != null) 'execution_id': executionId,
      if (componentId != null) 'component_id': componentId,
      if (property != null) 'property': property,
      if (status != null) 'status': status,
      if (narrative != null) 'narrative': narrative,
      if (source != null) 'source': source,
      if (verificationStatus != null) 'verification_status': verificationStatus,
      if (observedAt != null) 'observed_at': observedAt,
      if (createdAtDevice != null) 'created_at_device': createdAtDevice,
      if (syncStatus != null) 'sync_status': syncStatus,
      if (serverId != null) 'server_id': serverId,
      if (syncError != null) 'sync_error': syncError,
      if (rowid != null) 'rowid': rowid,
    });
  }

  LocalObservationsCompanion copyWith({
    Value<String>? clientEventId,
    Value<String>? executionId,
    Value<String?>? componentId,
    Value<String?>? property,
    Value<String?>? status,
    Value<String>? narrative,
    Value<String>? source,
    Value<String>? verificationStatus,
    Value<DateTime>? observedAt,
    Value<DateTime>? createdAtDevice,
    Value<String>? syncStatus,
    Value<String?>? serverId,
    Value<String?>? syncError,
    Value<int>? rowid,
  }) {
    return LocalObservationsCompanion(
      clientEventId: clientEventId ?? this.clientEventId,
      executionId: executionId ?? this.executionId,
      componentId: componentId ?? this.componentId,
      property: property ?? this.property,
      status: status ?? this.status,
      narrative: narrative ?? this.narrative,
      source: source ?? this.source,
      verificationStatus: verificationStatus ?? this.verificationStatus,
      observedAt: observedAt ?? this.observedAt,
      createdAtDevice: createdAtDevice ?? this.createdAtDevice,
      syncStatus: syncStatus ?? this.syncStatus,
      serverId: serverId ?? this.serverId,
      syncError: syncError ?? this.syncError,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (clientEventId.present) {
      map['client_event_id'] = Variable<String>(clientEventId.value);
    }
    if (executionId.present) {
      map['execution_id'] = Variable<String>(executionId.value);
    }
    if (componentId.present) {
      map['component_id'] = Variable<String>(componentId.value);
    }
    if (property.present) {
      map['property'] = Variable<String>(property.value);
    }
    if (status.present) {
      map['status'] = Variable<String>(status.value);
    }
    if (narrative.present) {
      map['narrative'] = Variable<String>(narrative.value);
    }
    if (source.present) {
      map['source'] = Variable<String>(source.value);
    }
    if (verificationStatus.present) {
      map['verification_status'] = Variable<String>(verificationStatus.value);
    }
    if (observedAt.present) {
      map['observed_at'] = Variable<DateTime>(observedAt.value);
    }
    if (createdAtDevice.present) {
      map['created_at_device'] = Variable<DateTime>(createdAtDevice.value);
    }
    if (syncStatus.present) {
      map['sync_status'] = Variable<String>(syncStatus.value);
    }
    if (serverId.present) {
      map['server_id'] = Variable<String>(serverId.value);
    }
    if (syncError.present) {
      map['sync_error'] = Variable<String>(syncError.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('LocalObservationsCompanion(')
          ..write('clientEventId: $clientEventId, ')
          ..write('executionId: $executionId, ')
          ..write('componentId: $componentId, ')
          ..write('property: $property, ')
          ..write('status: $status, ')
          ..write('narrative: $narrative, ')
          ..write('source: $source, ')
          ..write('verificationStatus: $verificationStatus, ')
          ..write('observedAt: $observedAt, ')
          ..write('createdAtDevice: $createdAtDevice, ')
          ..write('syncStatus: $syncStatus, ')
          ..write('serverId: $serverId, ')
          ..write('syncError: $syncError, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $LocalAttachmentsTable extends LocalAttachments
    with TableInfo<$LocalAttachmentsTable, LocalAttachment> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $LocalAttachmentsTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _clientEventIdMeta = const VerificationMeta(
    'clientEventId',
  );
  @override
  late final GeneratedColumn<String> clientEventId = GeneratedColumn<String>(
    'client_event_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _entityKindMeta = const VerificationMeta(
    'entityKind',
  );
  @override
  late final GeneratedColumn<String> entityKind = GeneratedColumn<String>(
    'entity_kind',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _entityIdMeta = const VerificationMeta(
    'entityId',
  );
  @override
  late final GeneratedColumn<String> entityId = GeneratedColumn<String>(
    'entity_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _siteIdMeta = const VerificationMeta('siteId');
  @override
  late final GeneratedColumn<String> siteId = GeneratedColumn<String>(
    'site_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _localPathMeta = const VerificationMeta(
    'localPath',
  );
  @override
  late final GeneratedColumn<String> localPath = GeneratedColumn<String>(
    'local_path',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _filenameMeta = const VerificationMeta(
    'filename',
  );
  @override
  late final GeneratedColumn<String> filename = GeneratedColumn<String>(
    'filename',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _mimeTypeMeta = const VerificationMeta(
    'mimeType',
  );
  @override
  late final GeneratedColumn<String> mimeType = GeneratedColumn<String>(
    'mime_type',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _sizeBytesMeta = const VerificationMeta(
    'sizeBytes',
  );
  @override
  late final GeneratedColumn<int> sizeBytes = GeneratedColumn<int>(
    'size_bytes',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _checksumSha256Meta = const VerificationMeta(
    'checksumSha256',
  );
  @override
  late final GeneratedColumn<String> checksumSha256 = GeneratedColumn<String>(
    'checksum_sha256',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _syncStatusMeta = const VerificationMeta(
    'syncStatus',
  );
  @override
  late final GeneratedColumn<String> syncStatus = GeneratedColumn<String>(
    'sync_status',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
    defaultValue: const Constant('PENDING'),
  );
  static const VerificationMeta _serverIdMeta = const VerificationMeta(
    'serverId',
  );
  @override
  late final GeneratedColumn<String> serverId = GeneratedColumn<String>(
    'server_id',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _syncErrorMeta = const VerificationMeta(
    'syncError',
  );
  @override
  late final GeneratedColumn<String> syncError = GeneratedColumn<String>(
    'sync_error',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _createdAtDeviceMeta = const VerificationMeta(
    'createdAtDevice',
  );
  @override
  late final GeneratedColumn<DateTime> createdAtDevice =
      GeneratedColumn<DateTime>(
        'created_at_device',
        aliasedName,
        false,
        type: DriftSqlType.dateTime,
        requiredDuringInsert: true,
      );
  @override
  List<GeneratedColumn> get $columns => [
    clientEventId,
    entityKind,
    entityId,
    siteId,
    localPath,
    filename,
    mimeType,
    sizeBytes,
    checksumSha256,
    syncStatus,
    serverId,
    syncError,
    createdAtDevice,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'local_attachments';
  @override
  VerificationContext validateIntegrity(
    Insertable<LocalAttachment> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('client_event_id')) {
      context.handle(
        _clientEventIdMeta,
        clientEventId.isAcceptableOrUnknown(
          data['client_event_id']!,
          _clientEventIdMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_clientEventIdMeta);
    }
    if (data.containsKey('entity_kind')) {
      context.handle(
        _entityKindMeta,
        entityKind.isAcceptableOrUnknown(data['entity_kind']!, _entityKindMeta),
      );
    } else if (isInserting) {
      context.missing(_entityKindMeta);
    }
    if (data.containsKey('entity_id')) {
      context.handle(
        _entityIdMeta,
        entityId.isAcceptableOrUnknown(data['entity_id']!, _entityIdMeta),
      );
    } else if (isInserting) {
      context.missing(_entityIdMeta);
    }
    if (data.containsKey('site_id')) {
      context.handle(
        _siteIdMeta,
        siteId.isAcceptableOrUnknown(data['site_id']!, _siteIdMeta),
      );
    } else if (isInserting) {
      context.missing(_siteIdMeta);
    }
    if (data.containsKey('local_path')) {
      context.handle(
        _localPathMeta,
        localPath.isAcceptableOrUnknown(data['local_path']!, _localPathMeta),
      );
    } else if (isInserting) {
      context.missing(_localPathMeta);
    }
    if (data.containsKey('filename')) {
      context.handle(
        _filenameMeta,
        filename.isAcceptableOrUnknown(data['filename']!, _filenameMeta),
      );
    } else if (isInserting) {
      context.missing(_filenameMeta);
    }
    if (data.containsKey('mime_type')) {
      context.handle(
        _mimeTypeMeta,
        mimeType.isAcceptableOrUnknown(data['mime_type']!, _mimeTypeMeta),
      );
    } else if (isInserting) {
      context.missing(_mimeTypeMeta);
    }
    if (data.containsKey('size_bytes')) {
      context.handle(
        _sizeBytesMeta,
        sizeBytes.isAcceptableOrUnknown(data['size_bytes']!, _sizeBytesMeta),
      );
    } else if (isInserting) {
      context.missing(_sizeBytesMeta);
    }
    if (data.containsKey('checksum_sha256')) {
      context.handle(
        _checksumSha256Meta,
        checksumSha256.isAcceptableOrUnknown(
          data['checksum_sha256']!,
          _checksumSha256Meta,
        ),
      );
    } else if (isInserting) {
      context.missing(_checksumSha256Meta);
    }
    if (data.containsKey('sync_status')) {
      context.handle(
        _syncStatusMeta,
        syncStatus.isAcceptableOrUnknown(data['sync_status']!, _syncStatusMeta),
      );
    }
    if (data.containsKey('server_id')) {
      context.handle(
        _serverIdMeta,
        serverId.isAcceptableOrUnknown(data['server_id']!, _serverIdMeta),
      );
    }
    if (data.containsKey('sync_error')) {
      context.handle(
        _syncErrorMeta,
        syncError.isAcceptableOrUnknown(data['sync_error']!, _syncErrorMeta),
      );
    }
    if (data.containsKey('created_at_device')) {
      context.handle(
        _createdAtDeviceMeta,
        createdAtDevice.isAcceptableOrUnknown(
          data['created_at_device']!,
          _createdAtDeviceMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_createdAtDeviceMeta);
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {clientEventId};
  @override
  LocalAttachment map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return LocalAttachment(
      clientEventId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}client_event_id'],
      )!,
      entityKind: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}entity_kind'],
      )!,
      entityId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}entity_id'],
      )!,
      siteId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}site_id'],
      )!,
      localPath: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}local_path'],
      )!,
      filename: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}filename'],
      )!,
      mimeType: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}mime_type'],
      )!,
      sizeBytes: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}size_bytes'],
      )!,
      checksumSha256: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}checksum_sha256'],
      )!,
      syncStatus: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}sync_status'],
      )!,
      serverId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}server_id'],
      ),
      syncError: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}sync_error'],
      ),
      createdAtDevice: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}created_at_device'],
      )!,
    );
  }

  @override
  $LocalAttachmentsTable createAlias(String alias) {
    return $LocalAttachmentsTable(attachedDatabase, alias);
  }
}

class LocalAttachment extends DataClass implements Insertable<LocalAttachment> {
  final String clientEventId;
  final String entityKind;
  final String entityId;
  final String siteId;
  final String localPath;
  final String filename;
  final String mimeType;
  final int sizeBytes;
  final String checksumSha256;
  final String syncStatus;
  final String? serverId;
  final String? syncError;
  final DateTime createdAtDevice;
  const LocalAttachment({
    required this.clientEventId,
    required this.entityKind,
    required this.entityId,
    required this.siteId,
    required this.localPath,
    required this.filename,
    required this.mimeType,
    required this.sizeBytes,
    required this.checksumSha256,
    required this.syncStatus,
    this.serverId,
    this.syncError,
    required this.createdAtDevice,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['client_event_id'] = Variable<String>(clientEventId);
    map['entity_kind'] = Variable<String>(entityKind);
    map['entity_id'] = Variable<String>(entityId);
    map['site_id'] = Variable<String>(siteId);
    map['local_path'] = Variable<String>(localPath);
    map['filename'] = Variable<String>(filename);
    map['mime_type'] = Variable<String>(mimeType);
    map['size_bytes'] = Variable<int>(sizeBytes);
    map['checksum_sha256'] = Variable<String>(checksumSha256);
    map['sync_status'] = Variable<String>(syncStatus);
    if (!nullToAbsent || serverId != null) {
      map['server_id'] = Variable<String>(serverId);
    }
    if (!nullToAbsent || syncError != null) {
      map['sync_error'] = Variable<String>(syncError);
    }
    map['created_at_device'] = Variable<DateTime>(createdAtDevice);
    return map;
  }

  LocalAttachmentsCompanion toCompanion(bool nullToAbsent) {
    return LocalAttachmentsCompanion(
      clientEventId: Value(clientEventId),
      entityKind: Value(entityKind),
      entityId: Value(entityId),
      siteId: Value(siteId),
      localPath: Value(localPath),
      filename: Value(filename),
      mimeType: Value(mimeType),
      sizeBytes: Value(sizeBytes),
      checksumSha256: Value(checksumSha256),
      syncStatus: Value(syncStatus),
      serverId: serverId == null && nullToAbsent
          ? const Value.absent()
          : Value(serverId),
      syncError: syncError == null && nullToAbsent
          ? const Value.absent()
          : Value(syncError),
      createdAtDevice: Value(createdAtDevice),
    );
  }

  factory LocalAttachment.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return LocalAttachment(
      clientEventId: serializer.fromJson<String>(json['clientEventId']),
      entityKind: serializer.fromJson<String>(json['entityKind']),
      entityId: serializer.fromJson<String>(json['entityId']),
      siteId: serializer.fromJson<String>(json['siteId']),
      localPath: serializer.fromJson<String>(json['localPath']),
      filename: serializer.fromJson<String>(json['filename']),
      mimeType: serializer.fromJson<String>(json['mimeType']),
      sizeBytes: serializer.fromJson<int>(json['sizeBytes']),
      checksumSha256: serializer.fromJson<String>(json['checksumSha256']),
      syncStatus: serializer.fromJson<String>(json['syncStatus']),
      serverId: serializer.fromJson<String?>(json['serverId']),
      syncError: serializer.fromJson<String?>(json['syncError']),
      createdAtDevice: serializer.fromJson<DateTime>(json['createdAtDevice']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'clientEventId': serializer.toJson<String>(clientEventId),
      'entityKind': serializer.toJson<String>(entityKind),
      'entityId': serializer.toJson<String>(entityId),
      'siteId': serializer.toJson<String>(siteId),
      'localPath': serializer.toJson<String>(localPath),
      'filename': serializer.toJson<String>(filename),
      'mimeType': serializer.toJson<String>(mimeType),
      'sizeBytes': serializer.toJson<int>(sizeBytes),
      'checksumSha256': serializer.toJson<String>(checksumSha256),
      'syncStatus': serializer.toJson<String>(syncStatus),
      'serverId': serializer.toJson<String?>(serverId),
      'syncError': serializer.toJson<String?>(syncError),
      'createdAtDevice': serializer.toJson<DateTime>(createdAtDevice),
    };
  }

  LocalAttachment copyWith({
    String? clientEventId,
    String? entityKind,
    String? entityId,
    String? siteId,
    String? localPath,
    String? filename,
    String? mimeType,
    int? sizeBytes,
    String? checksumSha256,
    String? syncStatus,
    Value<String?> serverId = const Value.absent(),
    Value<String?> syncError = const Value.absent(),
    DateTime? createdAtDevice,
  }) => LocalAttachment(
    clientEventId: clientEventId ?? this.clientEventId,
    entityKind: entityKind ?? this.entityKind,
    entityId: entityId ?? this.entityId,
    siteId: siteId ?? this.siteId,
    localPath: localPath ?? this.localPath,
    filename: filename ?? this.filename,
    mimeType: mimeType ?? this.mimeType,
    sizeBytes: sizeBytes ?? this.sizeBytes,
    checksumSha256: checksumSha256 ?? this.checksumSha256,
    syncStatus: syncStatus ?? this.syncStatus,
    serverId: serverId.present ? serverId.value : this.serverId,
    syncError: syncError.present ? syncError.value : this.syncError,
    createdAtDevice: createdAtDevice ?? this.createdAtDevice,
  );
  LocalAttachment copyWithCompanion(LocalAttachmentsCompanion data) {
    return LocalAttachment(
      clientEventId: data.clientEventId.present
          ? data.clientEventId.value
          : this.clientEventId,
      entityKind: data.entityKind.present
          ? data.entityKind.value
          : this.entityKind,
      entityId: data.entityId.present ? data.entityId.value : this.entityId,
      siteId: data.siteId.present ? data.siteId.value : this.siteId,
      localPath: data.localPath.present ? data.localPath.value : this.localPath,
      filename: data.filename.present ? data.filename.value : this.filename,
      mimeType: data.mimeType.present ? data.mimeType.value : this.mimeType,
      sizeBytes: data.sizeBytes.present ? data.sizeBytes.value : this.sizeBytes,
      checksumSha256: data.checksumSha256.present
          ? data.checksumSha256.value
          : this.checksumSha256,
      syncStatus: data.syncStatus.present
          ? data.syncStatus.value
          : this.syncStatus,
      serverId: data.serverId.present ? data.serverId.value : this.serverId,
      syncError: data.syncError.present ? data.syncError.value : this.syncError,
      createdAtDevice: data.createdAtDevice.present
          ? data.createdAtDevice.value
          : this.createdAtDevice,
    );
  }

  @override
  String toString() {
    return (StringBuffer('LocalAttachment(')
          ..write('clientEventId: $clientEventId, ')
          ..write('entityKind: $entityKind, ')
          ..write('entityId: $entityId, ')
          ..write('siteId: $siteId, ')
          ..write('localPath: $localPath, ')
          ..write('filename: $filename, ')
          ..write('mimeType: $mimeType, ')
          ..write('sizeBytes: $sizeBytes, ')
          ..write('checksumSha256: $checksumSha256, ')
          ..write('syncStatus: $syncStatus, ')
          ..write('serverId: $serverId, ')
          ..write('syncError: $syncError, ')
          ..write('createdAtDevice: $createdAtDevice')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(
    clientEventId,
    entityKind,
    entityId,
    siteId,
    localPath,
    filename,
    mimeType,
    sizeBytes,
    checksumSha256,
    syncStatus,
    serverId,
    syncError,
    createdAtDevice,
  );
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is LocalAttachment &&
          other.clientEventId == this.clientEventId &&
          other.entityKind == this.entityKind &&
          other.entityId == this.entityId &&
          other.siteId == this.siteId &&
          other.localPath == this.localPath &&
          other.filename == this.filename &&
          other.mimeType == this.mimeType &&
          other.sizeBytes == this.sizeBytes &&
          other.checksumSha256 == this.checksumSha256 &&
          other.syncStatus == this.syncStatus &&
          other.serverId == this.serverId &&
          other.syncError == this.syncError &&
          other.createdAtDevice == this.createdAtDevice);
}

class LocalAttachmentsCompanion extends UpdateCompanion<LocalAttachment> {
  final Value<String> clientEventId;
  final Value<String> entityKind;
  final Value<String> entityId;
  final Value<String> siteId;
  final Value<String> localPath;
  final Value<String> filename;
  final Value<String> mimeType;
  final Value<int> sizeBytes;
  final Value<String> checksumSha256;
  final Value<String> syncStatus;
  final Value<String?> serverId;
  final Value<String?> syncError;
  final Value<DateTime> createdAtDevice;
  final Value<int> rowid;
  const LocalAttachmentsCompanion({
    this.clientEventId = const Value.absent(),
    this.entityKind = const Value.absent(),
    this.entityId = const Value.absent(),
    this.siteId = const Value.absent(),
    this.localPath = const Value.absent(),
    this.filename = const Value.absent(),
    this.mimeType = const Value.absent(),
    this.sizeBytes = const Value.absent(),
    this.checksumSha256 = const Value.absent(),
    this.syncStatus = const Value.absent(),
    this.serverId = const Value.absent(),
    this.syncError = const Value.absent(),
    this.createdAtDevice = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  LocalAttachmentsCompanion.insert({
    required String clientEventId,
    required String entityKind,
    required String entityId,
    required String siteId,
    required String localPath,
    required String filename,
    required String mimeType,
    required int sizeBytes,
    required String checksumSha256,
    this.syncStatus = const Value.absent(),
    this.serverId = const Value.absent(),
    this.syncError = const Value.absent(),
    required DateTime createdAtDevice,
    this.rowid = const Value.absent(),
  }) : clientEventId = Value(clientEventId),
       entityKind = Value(entityKind),
       entityId = Value(entityId),
       siteId = Value(siteId),
       localPath = Value(localPath),
       filename = Value(filename),
       mimeType = Value(mimeType),
       sizeBytes = Value(sizeBytes),
       checksumSha256 = Value(checksumSha256),
       createdAtDevice = Value(createdAtDevice);
  static Insertable<LocalAttachment> custom({
    Expression<String>? clientEventId,
    Expression<String>? entityKind,
    Expression<String>? entityId,
    Expression<String>? siteId,
    Expression<String>? localPath,
    Expression<String>? filename,
    Expression<String>? mimeType,
    Expression<int>? sizeBytes,
    Expression<String>? checksumSha256,
    Expression<String>? syncStatus,
    Expression<String>? serverId,
    Expression<String>? syncError,
    Expression<DateTime>? createdAtDevice,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (clientEventId != null) 'client_event_id': clientEventId,
      if (entityKind != null) 'entity_kind': entityKind,
      if (entityId != null) 'entity_id': entityId,
      if (siteId != null) 'site_id': siteId,
      if (localPath != null) 'local_path': localPath,
      if (filename != null) 'filename': filename,
      if (mimeType != null) 'mime_type': mimeType,
      if (sizeBytes != null) 'size_bytes': sizeBytes,
      if (checksumSha256 != null) 'checksum_sha256': checksumSha256,
      if (syncStatus != null) 'sync_status': syncStatus,
      if (serverId != null) 'server_id': serverId,
      if (syncError != null) 'sync_error': syncError,
      if (createdAtDevice != null) 'created_at_device': createdAtDevice,
      if (rowid != null) 'rowid': rowid,
    });
  }

  LocalAttachmentsCompanion copyWith({
    Value<String>? clientEventId,
    Value<String>? entityKind,
    Value<String>? entityId,
    Value<String>? siteId,
    Value<String>? localPath,
    Value<String>? filename,
    Value<String>? mimeType,
    Value<int>? sizeBytes,
    Value<String>? checksumSha256,
    Value<String>? syncStatus,
    Value<String?>? serverId,
    Value<String?>? syncError,
    Value<DateTime>? createdAtDevice,
    Value<int>? rowid,
  }) {
    return LocalAttachmentsCompanion(
      clientEventId: clientEventId ?? this.clientEventId,
      entityKind: entityKind ?? this.entityKind,
      entityId: entityId ?? this.entityId,
      siteId: siteId ?? this.siteId,
      localPath: localPath ?? this.localPath,
      filename: filename ?? this.filename,
      mimeType: mimeType ?? this.mimeType,
      sizeBytes: sizeBytes ?? this.sizeBytes,
      checksumSha256: checksumSha256 ?? this.checksumSha256,
      syncStatus: syncStatus ?? this.syncStatus,
      serverId: serverId ?? this.serverId,
      syncError: syncError ?? this.syncError,
      createdAtDevice: createdAtDevice ?? this.createdAtDevice,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (clientEventId.present) {
      map['client_event_id'] = Variable<String>(clientEventId.value);
    }
    if (entityKind.present) {
      map['entity_kind'] = Variable<String>(entityKind.value);
    }
    if (entityId.present) {
      map['entity_id'] = Variable<String>(entityId.value);
    }
    if (siteId.present) {
      map['site_id'] = Variable<String>(siteId.value);
    }
    if (localPath.present) {
      map['local_path'] = Variable<String>(localPath.value);
    }
    if (filename.present) {
      map['filename'] = Variable<String>(filename.value);
    }
    if (mimeType.present) {
      map['mime_type'] = Variable<String>(mimeType.value);
    }
    if (sizeBytes.present) {
      map['size_bytes'] = Variable<int>(sizeBytes.value);
    }
    if (checksumSha256.present) {
      map['checksum_sha256'] = Variable<String>(checksumSha256.value);
    }
    if (syncStatus.present) {
      map['sync_status'] = Variable<String>(syncStatus.value);
    }
    if (serverId.present) {
      map['server_id'] = Variable<String>(serverId.value);
    }
    if (syncError.present) {
      map['sync_error'] = Variable<String>(syncError.value);
    }
    if (createdAtDevice.present) {
      map['created_at_device'] = Variable<DateTime>(createdAtDevice.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('LocalAttachmentsCompanion(')
          ..write('clientEventId: $clientEventId, ')
          ..write('entityKind: $entityKind, ')
          ..write('entityId: $entityId, ')
          ..write('siteId: $siteId, ')
          ..write('localPath: $localPath, ')
          ..write('filename: $filename, ')
          ..write('mimeType: $mimeType, ')
          ..write('sizeBytes: $sizeBytes, ')
          ..write('checksumSha256: $checksumSha256, ')
          ..write('syncStatus: $syncStatus, ')
          ..write('serverId: $serverId, ')
          ..write('syncError: $syncError, ')
          ..write('createdAtDevice: $createdAtDevice, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $OutboxEntriesTable extends OutboxEntries
    with TableInfo<$OutboxEntriesTable, OutboxEntry> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $OutboxEntriesTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _idMeta = const VerificationMeta('id');
  @override
  late final GeneratedColumn<String> id = GeneratedColumn<String>(
    'id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _clientEventIdMeta = const VerificationMeta(
    'clientEventId',
  );
  @override
  late final GeneratedColumn<String> clientEventId = GeneratedColumn<String>(
    'client_event_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _idempotencyKeyMeta = const VerificationMeta(
    'idempotencyKey',
  );
  @override
  late final GeneratedColumn<String> idempotencyKey = GeneratedColumn<String>(
    'idempotency_key',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _operationMeta = const VerificationMeta(
    'operation',
  );
  @override
  late final GeneratedColumn<String> operation = GeneratedColumn<String>(
    'operation',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _pathMeta = const VerificationMeta('path');
  @override
  late final GeneratedColumn<String> path = GeneratedColumn<String>(
    'path',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _payloadJsonMeta = const VerificationMeta(
    'payloadJson',
  );
  @override
  late final GeneratedColumn<String> payloadJson = GeneratedColumn<String>(
    'payload_json',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _statusMeta = const VerificationMeta('status');
  @override
  late final GeneratedColumn<String> status = GeneratedColumn<String>(
    'status',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
    defaultValue: const Constant('PENDING'),
  );
  static const VerificationMeta _attemptCountMeta = const VerificationMeta(
    'attemptCount',
  );
  @override
  late final GeneratedColumn<int> attemptCount = GeneratedColumn<int>(
    'attempt_count',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: false,
    defaultValue: const Constant(0),
  );
  static const VerificationMeta _nextAttemptAtMeta = const VerificationMeta(
    'nextAttemptAt',
  );
  @override
  late final GeneratedColumn<DateTime> nextAttemptAt =
      GeneratedColumn<DateTime>(
        'next_attempt_at',
        aliasedName,
        false,
        type: DriftSqlType.dateTime,
        requiredDuringInsert: true,
      );
  static const VerificationMeta _createdAtDeviceMeta = const VerificationMeta(
    'createdAtDevice',
  );
  @override
  late final GeneratedColumn<DateTime> createdAtDevice =
      GeneratedColumn<DateTime>(
        'created_at_device',
        aliasedName,
        false,
        type: DriftSqlType.dateTime,
        requiredDuringInsert: true,
      );
  static const VerificationMeta _lastAttemptAtMeta = const VerificationMeta(
    'lastAttemptAt',
  );
  @override
  late final GeneratedColumn<DateTime> lastAttemptAt =
      GeneratedColumn<DateTime>(
        'last_attempt_at',
        aliasedName,
        true,
        type: DriftSqlType.dateTime,
        requiredDuringInsert: false,
      );
  static const VerificationMeta _lastErrorMeta = const VerificationMeta(
    'lastError',
  );
  @override
  late final GeneratedColumn<String> lastError = GeneratedColumn<String>(
    'last_error',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _baseServerVersionMeta = const VerificationMeta(
    'baseServerVersion',
  );
  @override
  late final GeneratedColumn<int> baseServerVersion = GeneratedColumn<int>(
    'base_server_version',
    aliasedName,
    true,
    type: DriftSqlType.int,
    requiredDuringInsert: false,
  );
  @override
  List<GeneratedColumn> get $columns => [
    id,
    clientEventId,
    idempotencyKey,
    operation,
    path,
    payloadJson,
    status,
    attemptCount,
    nextAttemptAt,
    createdAtDevice,
    lastAttemptAt,
    lastError,
    baseServerVersion,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'outbox_entries';
  @override
  VerificationContext validateIntegrity(
    Insertable<OutboxEntry> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('id')) {
      context.handle(_idMeta, id.isAcceptableOrUnknown(data['id']!, _idMeta));
    } else if (isInserting) {
      context.missing(_idMeta);
    }
    if (data.containsKey('client_event_id')) {
      context.handle(
        _clientEventIdMeta,
        clientEventId.isAcceptableOrUnknown(
          data['client_event_id']!,
          _clientEventIdMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_clientEventIdMeta);
    }
    if (data.containsKey('idempotency_key')) {
      context.handle(
        _idempotencyKeyMeta,
        idempotencyKey.isAcceptableOrUnknown(
          data['idempotency_key']!,
          _idempotencyKeyMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_idempotencyKeyMeta);
    }
    if (data.containsKey('operation')) {
      context.handle(
        _operationMeta,
        operation.isAcceptableOrUnknown(data['operation']!, _operationMeta),
      );
    } else if (isInserting) {
      context.missing(_operationMeta);
    }
    if (data.containsKey('path')) {
      context.handle(
        _pathMeta,
        path.isAcceptableOrUnknown(data['path']!, _pathMeta),
      );
    } else if (isInserting) {
      context.missing(_pathMeta);
    }
    if (data.containsKey('payload_json')) {
      context.handle(
        _payloadJsonMeta,
        payloadJson.isAcceptableOrUnknown(
          data['payload_json']!,
          _payloadJsonMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_payloadJsonMeta);
    }
    if (data.containsKey('status')) {
      context.handle(
        _statusMeta,
        status.isAcceptableOrUnknown(data['status']!, _statusMeta),
      );
    }
    if (data.containsKey('attempt_count')) {
      context.handle(
        _attemptCountMeta,
        attemptCount.isAcceptableOrUnknown(
          data['attempt_count']!,
          _attemptCountMeta,
        ),
      );
    }
    if (data.containsKey('next_attempt_at')) {
      context.handle(
        _nextAttemptAtMeta,
        nextAttemptAt.isAcceptableOrUnknown(
          data['next_attempt_at']!,
          _nextAttemptAtMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_nextAttemptAtMeta);
    }
    if (data.containsKey('created_at_device')) {
      context.handle(
        _createdAtDeviceMeta,
        createdAtDevice.isAcceptableOrUnknown(
          data['created_at_device']!,
          _createdAtDeviceMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_createdAtDeviceMeta);
    }
    if (data.containsKey('last_attempt_at')) {
      context.handle(
        _lastAttemptAtMeta,
        lastAttemptAt.isAcceptableOrUnknown(
          data['last_attempt_at']!,
          _lastAttemptAtMeta,
        ),
      );
    }
    if (data.containsKey('last_error')) {
      context.handle(
        _lastErrorMeta,
        lastError.isAcceptableOrUnknown(data['last_error']!, _lastErrorMeta),
      );
    }
    if (data.containsKey('base_server_version')) {
      context.handle(
        _baseServerVersionMeta,
        baseServerVersion.isAcceptableOrUnknown(
          data['base_server_version']!,
          _baseServerVersionMeta,
        ),
      );
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {id};
  @override
  List<Set<GeneratedColumn>> get uniqueKeys => [
    {operation, clientEventId},
  ];
  @override
  OutboxEntry map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return OutboxEntry(
      id: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}id'],
      )!,
      clientEventId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}client_event_id'],
      )!,
      idempotencyKey: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}idempotency_key'],
      )!,
      operation: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}operation'],
      )!,
      path: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}path'],
      )!,
      payloadJson: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}payload_json'],
      )!,
      status: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}status'],
      )!,
      attemptCount: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}attempt_count'],
      )!,
      nextAttemptAt: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}next_attempt_at'],
      )!,
      createdAtDevice: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}created_at_device'],
      )!,
      lastAttemptAt: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}last_attempt_at'],
      ),
      lastError: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}last_error'],
      ),
      baseServerVersion: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}base_server_version'],
      ),
    );
  }

  @override
  $OutboxEntriesTable createAlias(String alias) {
    return $OutboxEntriesTable(attachedDatabase, alias);
  }
}

class OutboxEntry extends DataClass implements Insertable<OutboxEntry> {
  final String id;
  final String clientEventId;
  final String idempotencyKey;
  final String operation;
  final String path;
  final String payloadJson;
  final String status;
  final int attemptCount;
  final DateTime nextAttemptAt;
  final DateTime createdAtDevice;
  final DateTime? lastAttemptAt;
  final String? lastError;
  final int? baseServerVersion;
  const OutboxEntry({
    required this.id,
    required this.clientEventId,
    required this.idempotencyKey,
    required this.operation,
    required this.path,
    required this.payloadJson,
    required this.status,
    required this.attemptCount,
    required this.nextAttemptAt,
    required this.createdAtDevice,
    this.lastAttemptAt,
    this.lastError,
    this.baseServerVersion,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['id'] = Variable<String>(id);
    map['client_event_id'] = Variable<String>(clientEventId);
    map['idempotency_key'] = Variable<String>(idempotencyKey);
    map['operation'] = Variable<String>(operation);
    map['path'] = Variable<String>(path);
    map['payload_json'] = Variable<String>(payloadJson);
    map['status'] = Variable<String>(status);
    map['attempt_count'] = Variable<int>(attemptCount);
    map['next_attempt_at'] = Variable<DateTime>(nextAttemptAt);
    map['created_at_device'] = Variable<DateTime>(createdAtDevice);
    if (!nullToAbsent || lastAttemptAt != null) {
      map['last_attempt_at'] = Variable<DateTime>(lastAttemptAt);
    }
    if (!nullToAbsent || lastError != null) {
      map['last_error'] = Variable<String>(lastError);
    }
    if (!nullToAbsent || baseServerVersion != null) {
      map['base_server_version'] = Variable<int>(baseServerVersion);
    }
    return map;
  }

  OutboxEntriesCompanion toCompanion(bool nullToAbsent) {
    return OutboxEntriesCompanion(
      id: Value(id),
      clientEventId: Value(clientEventId),
      idempotencyKey: Value(idempotencyKey),
      operation: Value(operation),
      path: Value(path),
      payloadJson: Value(payloadJson),
      status: Value(status),
      attemptCount: Value(attemptCount),
      nextAttemptAt: Value(nextAttemptAt),
      createdAtDevice: Value(createdAtDevice),
      lastAttemptAt: lastAttemptAt == null && nullToAbsent
          ? const Value.absent()
          : Value(lastAttemptAt),
      lastError: lastError == null && nullToAbsent
          ? const Value.absent()
          : Value(lastError),
      baseServerVersion: baseServerVersion == null && nullToAbsent
          ? const Value.absent()
          : Value(baseServerVersion),
    );
  }

  factory OutboxEntry.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return OutboxEntry(
      id: serializer.fromJson<String>(json['id']),
      clientEventId: serializer.fromJson<String>(json['clientEventId']),
      idempotencyKey: serializer.fromJson<String>(json['idempotencyKey']),
      operation: serializer.fromJson<String>(json['operation']),
      path: serializer.fromJson<String>(json['path']),
      payloadJson: serializer.fromJson<String>(json['payloadJson']),
      status: serializer.fromJson<String>(json['status']),
      attemptCount: serializer.fromJson<int>(json['attemptCount']),
      nextAttemptAt: serializer.fromJson<DateTime>(json['nextAttemptAt']),
      createdAtDevice: serializer.fromJson<DateTime>(json['createdAtDevice']),
      lastAttemptAt: serializer.fromJson<DateTime?>(json['lastAttemptAt']),
      lastError: serializer.fromJson<String?>(json['lastError']),
      baseServerVersion: serializer.fromJson<int?>(json['baseServerVersion']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'id': serializer.toJson<String>(id),
      'clientEventId': serializer.toJson<String>(clientEventId),
      'idempotencyKey': serializer.toJson<String>(idempotencyKey),
      'operation': serializer.toJson<String>(operation),
      'path': serializer.toJson<String>(path),
      'payloadJson': serializer.toJson<String>(payloadJson),
      'status': serializer.toJson<String>(status),
      'attemptCount': serializer.toJson<int>(attemptCount),
      'nextAttemptAt': serializer.toJson<DateTime>(nextAttemptAt),
      'createdAtDevice': serializer.toJson<DateTime>(createdAtDevice),
      'lastAttemptAt': serializer.toJson<DateTime?>(lastAttemptAt),
      'lastError': serializer.toJson<String?>(lastError),
      'baseServerVersion': serializer.toJson<int?>(baseServerVersion),
    };
  }

  OutboxEntry copyWith({
    String? id,
    String? clientEventId,
    String? idempotencyKey,
    String? operation,
    String? path,
    String? payloadJson,
    String? status,
    int? attemptCount,
    DateTime? nextAttemptAt,
    DateTime? createdAtDevice,
    Value<DateTime?> lastAttemptAt = const Value.absent(),
    Value<String?> lastError = const Value.absent(),
    Value<int?> baseServerVersion = const Value.absent(),
  }) => OutboxEntry(
    id: id ?? this.id,
    clientEventId: clientEventId ?? this.clientEventId,
    idempotencyKey: idempotencyKey ?? this.idempotencyKey,
    operation: operation ?? this.operation,
    path: path ?? this.path,
    payloadJson: payloadJson ?? this.payloadJson,
    status: status ?? this.status,
    attemptCount: attemptCount ?? this.attemptCount,
    nextAttemptAt: nextAttemptAt ?? this.nextAttemptAt,
    createdAtDevice: createdAtDevice ?? this.createdAtDevice,
    lastAttemptAt: lastAttemptAt.present
        ? lastAttemptAt.value
        : this.lastAttemptAt,
    lastError: lastError.present ? lastError.value : this.lastError,
    baseServerVersion: baseServerVersion.present
        ? baseServerVersion.value
        : this.baseServerVersion,
  );
  OutboxEntry copyWithCompanion(OutboxEntriesCompanion data) {
    return OutboxEntry(
      id: data.id.present ? data.id.value : this.id,
      clientEventId: data.clientEventId.present
          ? data.clientEventId.value
          : this.clientEventId,
      idempotencyKey: data.idempotencyKey.present
          ? data.idempotencyKey.value
          : this.idempotencyKey,
      operation: data.operation.present ? data.operation.value : this.operation,
      path: data.path.present ? data.path.value : this.path,
      payloadJson: data.payloadJson.present
          ? data.payloadJson.value
          : this.payloadJson,
      status: data.status.present ? data.status.value : this.status,
      attemptCount: data.attemptCount.present
          ? data.attemptCount.value
          : this.attemptCount,
      nextAttemptAt: data.nextAttemptAt.present
          ? data.nextAttemptAt.value
          : this.nextAttemptAt,
      createdAtDevice: data.createdAtDevice.present
          ? data.createdAtDevice.value
          : this.createdAtDevice,
      lastAttemptAt: data.lastAttemptAt.present
          ? data.lastAttemptAt.value
          : this.lastAttemptAt,
      lastError: data.lastError.present ? data.lastError.value : this.lastError,
      baseServerVersion: data.baseServerVersion.present
          ? data.baseServerVersion.value
          : this.baseServerVersion,
    );
  }

  @override
  String toString() {
    return (StringBuffer('OutboxEntry(')
          ..write('id: $id, ')
          ..write('clientEventId: $clientEventId, ')
          ..write('idempotencyKey: $idempotencyKey, ')
          ..write('operation: $operation, ')
          ..write('path: $path, ')
          ..write('payloadJson: $payloadJson, ')
          ..write('status: $status, ')
          ..write('attemptCount: $attemptCount, ')
          ..write('nextAttemptAt: $nextAttemptAt, ')
          ..write('createdAtDevice: $createdAtDevice, ')
          ..write('lastAttemptAt: $lastAttemptAt, ')
          ..write('lastError: $lastError, ')
          ..write('baseServerVersion: $baseServerVersion')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(
    id,
    clientEventId,
    idempotencyKey,
    operation,
    path,
    payloadJson,
    status,
    attemptCount,
    nextAttemptAt,
    createdAtDevice,
    lastAttemptAt,
    lastError,
    baseServerVersion,
  );
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is OutboxEntry &&
          other.id == this.id &&
          other.clientEventId == this.clientEventId &&
          other.idempotencyKey == this.idempotencyKey &&
          other.operation == this.operation &&
          other.path == this.path &&
          other.payloadJson == this.payloadJson &&
          other.status == this.status &&
          other.attemptCount == this.attemptCount &&
          other.nextAttemptAt == this.nextAttemptAt &&
          other.createdAtDevice == this.createdAtDevice &&
          other.lastAttemptAt == this.lastAttemptAt &&
          other.lastError == this.lastError &&
          other.baseServerVersion == this.baseServerVersion);
}

class OutboxEntriesCompanion extends UpdateCompanion<OutboxEntry> {
  final Value<String> id;
  final Value<String> clientEventId;
  final Value<String> idempotencyKey;
  final Value<String> operation;
  final Value<String> path;
  final Value<String> payloadJson;
  final Value<String> status;
  final Value<int> attemptCount;
  final Value<DateTime> nextAttemptAt;
  final Value<DateTime> createdAtDevice;
  final Value<DateTime?> lastAttemptAt;
  final Value<String?> lastError;
  final Value<int?> baseServerVersion;
  final Value<int> rowid;
  const OutboxEntriesCompanion({
    this.id = const Value.absent(),
    this.clientEventId = const Value.absent(),
    this.idempotencyKey = const Value.absent(),
    this.operation = const Value.absent(),
    this.path = const Value.absent(),
    this.payloadJson = const Value.absent(),
    this.status = const Value.absent(),
    this.attemptCount = const Value.absent(),
    this.nextAttemptAt = const Value.absent(),
    this.createdAtDevice = const Value.absent(),
    this.lastAttemptAt = const Value.absent(),
    this.lastError = const Value.absent(),
    this.baseServerVersion = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  OutboxEntriesCompanion.insert({
    required String id,
    required String clientEventId,
    required String idempotencyKey,
    required String operation,
    required String path,
    required String payloadJson,
    this.status = const Value.absent(),
    this.attemptCount = const Value.absent(),
    required DateTime nextAttemptAt,
    required DateTime createdAtDevice,
    this.lastAttemptAt = const Value.absent(),
    this.lastError = const Value.absent(),
    this.baseServerVersion = const Value.absent(),
    this.rowid = const Value.absent(),
  }) : id = Value(id),
       clientEventId = Value(clientEventId),
       idempotencyKey = Value(idempotencyKey),
       operation = Value(operation),
       path = Value(path),
       payloadJson = Value(payloadJson),
       nextAttemptAt = Value(nextAttemptAt),
       createdAtDevice = Value(createdAtDevice);
  static Insertable<OutboxEntry> custom({
    Expression<String>? id,
    Expression<String>? clientEventId,
    Expression<String>? idempotencyKey,
    Expression<String>? operation,
    Expression<String>? path,
    Expression<String>? payloadJson,
    Expression<String>? status,
    Expression<int>? attemptCount,
    Expression<DateTime>? nextAttemptAt,
    Expression<DateTime>? createdAtDevice,
    Expression<DateTime>? lastAttemptAt,
    Expression<String>? lastError,
    Expression<int>? baseServerVersion,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (id != null) 'id': id,
      if (clientEventId != null) 'client_event_id': clientEventId,
      if (idempotencyKey != null) 'idempotency_key': idempotencyKey,
      if (operation != null) 'operation': operation,
      if (path != null) 'path': path,
      if (payloadJson != null) 'payload_json': payloadJson,
      if (status != null) 'status': status,
      if (attemptCount != null) 'attempt_count': attemptCount,
      if (nextAttemptAt != null) 'next_attempt_at': nextAttemptAt,
      if (createdAtDevice != null) 'created_at_device': createdAtDevice,
      if (lastAttemptAt != null) 'last_attempt_at': lastAttemptAt,
      if (lastError != null) 'last_error': lastError,
      if (baseServerVersion != null) 'base_server_version': baseServerVersion,
      if (rowid != null) 'rowid': rowid,
    });
  }

  OutboxEntriesCompanion copyWith({
    Value<String>? id,
    Value<String>? clientEventId,
    Value<String>? idempotencyKey,
    Value<String>? operation,
    Value<String>? path,
    Value<String>? payloadJson,
    Value<String>? status,
    Value<int>? attemptCount,
    Value<DateTime>? nextAttemptAt,
    Value<DateTime>? createdAtDevice,
    Value<DateTime?>? lastAttemptAt,
    Value<String?>? lastError,
    Value<int?>? baseServerVersion,
    Value<int>? rowid,
  }) {
    return OutboxEntriesCompanion(
      id: id ?? this.id,
      clientEventId: clientEventId ?? this.clientEventId,
      idempotencyKey: idempotencyKey ?? this.idempotencyKey,
      operation: operation ?? this.operation,
      path: path ?? this.path,
      payloadJson: payloadJson ?? this.payloadJson,
      status: status ?? this.status,
      attemptCount: attemptCount ?? this.attemptCount,
      nextAttemptAt: nextAttemptAt ?? this.nextAttemptAt,
      createdAtDevice: createdAtDevice ?? this.createdAtDevice,
      lastAttemptAt: lastAttemptAt ?? this.lastAttemptAt,
      lastError: lastError ?? this.lastError,
      baseServerVersion: baseServerVersion ?? this.baseServerVersion,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (id.present) {
      map['id'] = Variable<String>(id.value);
    }
    if (clientEventId.present) {
      map['client_event_id'] = Variable<String>(clientEventId.value);
    }
    if (idempotencyKey.present) {
      map['idempotency_key'] = Variable<String>(idempotencyKey.value);
    }
    if (operation.present) {
      map['operation'] = Variable<String>(operation.value);
    }
    if (path.present) {
      map['path'] = Variable<String>(path.value);
    }
    if (payloadJson.present) {
      map['payload_json'] = Variable<String>(payloadJson.value);
    }
    if (status.present) {
      map['status'] = Variable<String>(status.value);
    }
    if (attemptCount.present) {
      map['attempt_count'] = Variable<int>(attemptCount.value);
    }
    if (nextAttemptAt.present) {
      map['next_attempt_at'] = Variable<DateTime>(nextAttemptAt.value);
    }
    if (createdAtDevice.present) {
      map['created_at_device'] = Variable<DateTime>(createdAtDevice.value);
    }
    if (lastAttemptAt.present) {
      map['last_attempt_at'] = Variable<DateTime>(lastAttemptAt.value);
    }
    if (lastError.present) {
      map['last_error'] = Variable<String>(lastError.value);
    }
    if (baseServerVersion.present) {
      map['base_server_version'] = Variable<int>(baseServerVersion.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('OutboxEntriesCompanion(')
          ..write('id: $id, ')
          ..write('clientEventId: $clientEventId, ')
          ..write('idempotencyKey: $idempotencyKey, ')
          ..write('operation: $operation, ')
          ..write('path: $path, ')
          ..write('payloadJson: $payloadJson, ')
          ..write('status: $status, ')
          ..write('attemptCount: $attemptCount, ')
          ..write('nextAttemptAt: $nextAttemptAt, ')
          ..write('createdAtDevice: $createdAtDevice, ')
          ..write('lastAttemptAt: $lastAttemptAt, ')
          ..write('lastError: $lastError, ')
          ..write('baseServerVersion: $baseServerVersion, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

abstract class _$MaintenanceDatabase extends GeneratedDatabase {
  _$MaintenanceDatabase(QueryExecutor e) : super(e);
  $MaintenanceDatabaseManager get managers => $MaintenanceDatabaseManager(this);
  late final $CachedAssetsTable cachedAssets = $CachedAssetsTable(this);
  late final $CachedIncidentsTable cachedIncidents = $CachedIncidentsTable(
    this,
  );
  late final $CachedExecutionsTable cachedExecutions = $CachedExecutionsTable(
    this,
  );
  late final $CachedExecutionStepsTable cachedExecutionSteps =
      $CachedExecutionStepsTable(this);
  late final $LocalMeasurementsTable localMeasurements =
      $LocalMeasurementsTable(this);
  late final $LocalObservationsTable localObservations =
      $LocalObservationsTable(this);
  late final $LocalAttachmentsTable localAttachments = $LocalAttachmentsTable(
    this,
  );
  late final $OutboxEntriesTable outboxEntries = $OutboxEntriesTable(this);
  @override
  Iterable<TableInfo<Table, Object?>> get allTables =>
      allSchemaEntities.whereType<TableInfo<Table, Object?>>();
  @override
  List<DatabaseSchemaEntity> get allSchemaEntities => [
    cachedAssets,
    cachedIncidents,
    cachedExecutions,
    cachedExecutionSteps,
    localMeasurements,
    localObservations,
    localAttachments,
    outboxEntries,
  ];
}

typedef $$CachedAssetsTableCreateCompanionBuilder =
    CachedAssetsCompanion Function({
      required String id,
      required String siteId,
      required String tag,
      required String name,
      required String assetClass,
      required String sourceOfTruth,
      Value<String?> criticality,
      required int serverVersion,
      required DateTime cachedAt,
      Value<int> rowid,
    });
typedef $$CachedAssetsTableUpdateCompanionBuilder =
    CachedAssetsCompanion Function({
      Value<String> id,
      Value<String> siteId,
      Value<String> tag,
      Value<String> name,
      Value<String> assetClass,
      Value<String> sourceOfTruth,
      Value<String?> criticality,
      Value<int> serverVersion,
      Value<DateTime> cachedAt,
      Value<int> rowid,
    });

class $$CachedAssetsTableFilterComposer
    extends Composer<_$MaintenanceDatabase, $CachedAssetsTable> {
  $$CachedAssetsTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get siteId => $composableBuilder(
    column: $table.siteId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get tag => $composableBuilder(
    column: $table.tag,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get name => $composableBuilder(
    column: $table.name,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get assetClass => $composableBuilder(
    column: $table.assetClass,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get sourceOfTruth => $composableBuilder(
    column: $table.sourceOfTruth,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get criticality => $composableBuilder(
    column: $table.criticality,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get serverVersion => $composableBuilder(
    column: $table.serverVersion,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get cachedAt => $composableBuilder(
    column: $table.cachedAt,
    builder: (column) => ColumnFilters(column),
  );
}

class $$CachedAssetsTableOrderingComposer
    extends Composer<_$MaintenanceDatabase, $CachedAssetsTable> {
  $$CachedAssetsTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get siteId => $composableBuilder(
    column: $table.siteId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get tag => $composableBuilder(
    column: $table.tag,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get name => $composableBuilder(
    column: $table.name,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get assetClass => $composableBuilder(
    column: $table.assetClass,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get sourceOfTruth => $composableBuilder(
    column: $table.sourceOfTruth,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get criticality => $composableBuilder(
    column: $table.criticality,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get serverVersion => $composableBuilder(
    column: $table.serverVersion,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get cachedAt => $composableBuilder(
    column: $table.cachedAt,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$CachedAssetsTableAnnotationComposer
    extends Composer<_$MaintenanceDatabase, $CachedAssetsTable> {
  $$CachedAssetsTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get id =>
      $composableBuilder(column: $table.id, builder: (column) => column);

  GeneratedColumn<String> get siteId =>
      $composableBuilder(column: $table.siteId, builder: (column) => column);

  GeneratedColumn<String> get tag =>
      $composableBuilder(column: $table.tag, builder: (column) => column);

  GeneratedColumn<String> get name =>
      $composableBuilder(column: $table.name, builder: (column) => column);

  GeneratedColumn<String> get assetClass => $composableBuilder(
    column: $table.assetClass,
    builder: (column) => column,
  );

  GeneratedColumn<String> get sourceOfTruth => $composableBuilder(
    column: $table.sourceOfTruth,
    builder: (column) => column,
  );

  GeneratedColumn<String> get criticality => $composableBuilder(
    column: $table.criticality,
    builder: (column) => column,
  );

  GeneratedColumn<int> get serverVersion => $composableBuilder(
    column: $table.serverVersion,
    builder: (column) => column,
  );

  GeneratedColumn<DateTime> get cachedAt =>
      $composableBuilder(column: $table.cachedAt, builder: (column) => column);
}

class $$CachedAssetsTableTableManager
    extends
        RootTableManager<
          _$MaintenanceDatabase,
          $CachedAssetsTable,
          CachedAsset,
          $$CachedAssetsTableFilterComposer,
          $$CachedAssetsTableOrderingComposer,
          $$CachedAssetsTableAnnotationComposer,
          $$CachedAssetsTableCreateCompanionBuilder,
          $$CachedAssetsTableUpdateCompanionBuilder,
          (
            CachedAsset,
            BaseReferences<
              _$MaintenanceDatabase,
              $CachedAssetsTable,
              CachedAsset
            >,
          ),
          CachedAsset,
          PrefetchHooks Function()
        > {
  $$CachedAssetsTableTableManager(
    _$MaintenanceDatabase db,
    $CachedAssetsTable table,
  ) : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$CachedAssetsTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$CachedAssetsTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$CachedAssetsTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> id = const Value.absent(),
                Value<String> siteId = const Value.absent(),
                Value<String> tag = const Value.absent(),
                Value<String> name = const Value.absent(),
                Value<String> assetClass = const Value.absent(),
                Value<String> sourceOfTruth = const Value.absent(),
                Value<String?> criticality = const Value.absent(),
                Value<int> serverVersion = const Value.absent(),
                Value<DateTime> cachedAt = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => CachedAssetsCompanion(
                id: id,
                siteId: siteId,
                tag: tag,
                name: name,
                assetClass: assetClass,
                sourceOfTruth: sourceOfTruth,
                criticality: criticality,
                serverVersion: serverVersion,
                cachedAt: cachedAt,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String id,
                required String siteId,
                required String tag,
                required String name,
                required String assetClass,
                required String sourceOfTruth,
                Value<String?> criticality = const Value.absent(),
                required int serverVersion,
                required DateTime cachedAt,
                Value<int> rowid = const Value.absent(),
              }) => CachedAssetsCompanion.insert(
                id: id,
                siteId: siteId,
                tag: tag,
                name: name,
                assetClass: assetClass,
                sourceOfTruth: sourceOfTruth,
                criticality: criticality,
                serverVersion: serverVersion,
                cachedAt: cachedAt,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$CachedAssetsTableProcessedTableManager =
    ProcessedTableManager<
      _$MaintenanceDatabase,
      $CachedAssetsTable,
      CachedAsset,
      $$CachedAssetsTableFilterComposer,
      $$CachedAssetsTableOrderingComposer,
      $$CachedAssetsTableAnnotationComposer,
      $$CachedAssetsTableCreateCompanionBuilder,
      $$CachedAssetsTableUpdateCompanionBuilder,
      (
        CachedAsset,
        BaseReferences<_$MaintenanceDatabase, $CachedAssetsTable, CachedAsset>,
      ),
      CachedAsset,
      PrefetchHooks Function()
    >;
typedef $$CachedIncidentsTableCreateCompanionBuilder =
    CachedIncidentsCompanion Function({
      required String id,
      required String siteId,
      required String assetId,
      required String assetTag,
      required String number,
      required String summary,
      required String severity,
      required String state,
      required DateTime detectedAt,
      required int serverVersion,
      required DateTime cachedAt,
      Value<int> rowid,
    });
typedef $$CachedIncidentsTableUpdateCompanionBuilder =
    CachedIncidentsCompanion Function({
      Value<String> id,
      Value<String> siteId,
      Value<String> assetId,
      Value<String> assetTag,
      Value<String> number,
      Value<String> summary,
      Value<String> severity,
      Value<String> state,
      Value<DateTime> detectedAt,
      Value<int> serverVersion,
      Value<DateTime> cachedAt,
      Value<int> rowid,
    });

class $$CachedIncidentsTableFilterComposer
    extends Composer<_$MaintenanceDatabase, $CachedIncidentsTable> {
  $$CachedIncidentsTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get siteId => $composableBuilder(
    column: $table.siteId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get assetId => $composableBuilder(
    column: $table.assetId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get assetTag => $composableBuilder(
    column: $table.assetTag,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get number => $composableBuilder(
    column: $table.number,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get summary => $composableBuilder(
    column: $table.summary,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get severity => $composableBuilder(
    column: $table.severity,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get state => $composableBuilder(
    column: $table.state,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get detectedAt => $composableBuilder(
    column: $table.detectedAt,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get serverVersion => $composableBuilder(
    column: $table.serverVersion,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get cachedAt => $composableBuilder(
    column: $table.cachedAt,
    builder: (column) => ColumnFilters(column),
  );
}

class $$CachedIncidentsTableOrderingComposer
    extends Composer<_$MaintenanceDatabase, $CachedIncidentsTable> {
  $$CachedIncidentsTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get siteId => $composableBuilder(
    column: $table.siteId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get assetId => $composableBuilder(
    column: $table.assetId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get assetTag => $composableBuilder(
    column: $table.assetTag,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get number => $composableBuilder(
    column: $table.number,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get summary => $composableBuilder(
    column: $table.summary,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get severity => $composableBuilder(
    column: $table.severity,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get state => $composableBuilder(
    column: $table.state,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get detectedAt => $composableBuilder(
    column: $table.detectedAt,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get serverVersion => $composableBuilder(
    column: $table.serverVersion,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get cachedAt => $composableBuilder(
    column: $table.cachedAt,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$CachedIncidentsTableAnnotationComposer
    extends Composer<_$MaintenanceDatabase, $CachedIncidentsTable> {
  $$CachedIncidentsTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get id =>
      $composableBuilder(column: $table.id, builder: (column) => column);

  GeneratedColumn<String> get siteId =>
      $composableBuilder(column: $table.siteId, builder: (column) => column);

  GeneratedColumn<String> get assetId =>
      $composableBuilder(column: $table.assetId, builder: (column) => column);

  GeneratedColumn<String> get assetTag =>
      $composableBuilder(column: $table.assetTag, builder: (column) => column);

  GeneratedColumn<String> get number =>
      $composableBuilder(column: $table.number, builder: (column) => column);

  GeneratedColumn<String> get summary =>
      $composableBuilder(column: $table.summary, builder: (column) => column);

  GeneratedColumn<String> get severity =>
      $composableBuilder(column: $table.severity, builder: (column) => column);

  GeneratedColumn<String> get state =>
      $composableBuilder(column: $table.state, builder: (column) => column);

  GeneratedColumn<DateTime> get detectedAt => $composableBuilder(
    column: $table.detectedAt,
    builder: (column) => column,
  );

  GeneratedColumn<int> get serverVersion => $composableBuilder(
    column: $table.serverVersion,
    builder: (column) => column,
  );

  GeneratedColumn<DateTime> get cachedAt =>
      $composableBuilder(column: $table.cachedAt, builder: (column) => column);
}

class $$CachedIncidentsTableTableManager
    extends
        RootTableManager<
          _$MaintenanceDatabase,
          $CachedIncidentsTable,
          CachedIncident,
          $$CachedIncidentsTableFilterComposer,
          $$CachedIncidentsTableOrderingComposer,
          $$CachedIncidentsTableAnnotationComposer,
          $$CachedIncidentsTableCreateCompanionBuilder,
          $$CachedIncidentsTableUpdateCompanionBuilder,
          (
            CachedIncident,
            BaseReferences<
              _$MaintenanceDatabase,
              $CachedIncidentsTable,
              CachedIncident
            >,
          ),
          CachedIncident,
          PrefetchHooks Function()
        > {
  $$CachedIncidentsTableTableManager(
    _$MaintenanceDatabase db,
    $CachedIncidentsTable table,
  ) : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$CachedIncidentsTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$CachedIncidentsTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$CachedIncidentsTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> id = const Value.absent(),
                Value<String> siteId = const Value.absent(),
                Value<String> assetId = const Value.absent(),
                Value<String> assetTag = const Value.absent(),
                Value<String> number = const Value.absent(),
                Value<String> summary = const Value.absent(),
                Value<String> severity = const Value.absent(),
                Value<String> state = const Value.absent(),
                Value<DateTime> detectedAt = const Value.absent(),
                Value<int> serverVersion = const Value.absent(),
                Value<DateTime> cachedAt = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => CachedIncidentsCompanion(
                id: id,
                siteId: siteId,
                assetId: assetId,
                assetTag: assetTag,
                number: number,
                summary: summary,
                severity: severity,
                state: state,
                detectedAt: detectedAt,
                serverVersion: serverVersion,
                cachedAt: cachedAt,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String id,
                required String siteId,
                required String assetId,
                required String assetTag,
                required String number,
                required String summary,
                required String severity,
                required String state,
                required DateTime detectedAt,
                required int serverVersion,
                required DateTime cachedAt,
                Value<int> rowid = const Value.absent(),
              }) => CachedIncidentsCompanion.insert(
                id: id,
                siteId: siteId,
                assetId: assetId,
                assetTag: assetTag,
                number: number,
                summary: summary,
                severity: severity,
                state: state,
                detectedAt: detectedAt,
                serverVersion: serverVersion,
                cachedAt: cachedAt,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$CachedIncidentsTableProcessedTableManager =
    ProcessedTableManager<
      _$MaintenanceDatabase,
      $CachedIncidentsTable,
      CachedIncident,
      $$CachedIncidentsTableFilterComposer,
      $$CachedIncidentsTableOrderingComposer,
      $$CachedIncidentsTableAnnotationComposer,
      $$CachedIncidentsTableCreateCompanionBuilder,
      $$CachedIncidentsTableUpdateCompanionBuilder,
      (
        CachedIncident,
        BaseReferences<
          _$MaintenanceDatabase,
          $CachedIncidentsTable,
          CachedIncident
        >,
      ),
      CachedIncident,
      PrefetchHooks Function()
    >;
typedef $$CachedExecutionsTableCreateCompanionBuilder =
    CachedExecutionsCompanion Function({
      required String id,
      required String siteId,
      required String incidentId,
      required String assetId,
      required String assetTag,
      required String purpose,
      required String state,
      required int serverVersion,
      Value<String> verifiedPrerequisites,
      required DateTime cachedAt,
      Value<int> rowid,
    });
typedef $$CachedExecutionsTableUpdateCompanionBuilder =
    CachedExecutionsCompanion Function({
      Value<String> id,
      Value<String> siteId,
      Value<String> incidentId,
      Value<String> assetId,
      Value<String> assetTag,
      Value<String> purpose,
      Value<String> state,
      Value<int> serverVersion,
      Value<String> verifiedPrerequisites,
      Value<DateTime> cachedAt,
      Value<int> rowid,
    });

class $$CachedExecutionsTableFilterComposer
    extends Composer<_$MaintenanceDatabase, $CachedExecutionsTable> {
  $$CachedExecutionsTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get siteId => $composableBuilder(
    column: $table.siteId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get incidentId => $composableBuilder(
    column: $table.incidentId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get assetId => $composableBuilder(
    column: $table.assetId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get assetTag => $composableBuilder(
    column: $table.assetTag,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get purpose => $composableBuilder(
    column: $table.purpose,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get state => $composableBuilder(
    column: $table.state,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get serverVersion => $composableBuilder(
    column: $table.serverVersion,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get verifiedPrerequisites => $composableBuilder(
    column: $table.verifiedPrerequisites,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get cachedAt => $composableBuilder(
    column: $table.cachedAt,
    builder: (column) => ColumnFilters(column),
  );
}

class $$CachedExecutionsTableOrderingComposer
    extends Composer<_$MaintenanceDatabase, $CachedExecutionsTable> {
  $$CachedExecutionsTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get siteId => $composableBuilder(
    column: $table.siteId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get incidentId => $composableBuilder(
    column: $table.incidentId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get assetId => $composableBuilder(
    column: $table.assetId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get assetTag => $composableBuilder(
    column: $table.assetTag,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get purpose => $composableBuilder(
    column: $table.purpose,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get state => $composableBuilder(
    column: $table.state,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get serverVersion => $composableBuilder(
    column: $table.serverVersion,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get verifiedPrerequisites => $composableBuilder(
    column: $table.verifiedPrerequisites,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get cachedAt => $composableBuilder(
    column: $table.cachedAt,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$CachedExecutionsTableAnnotationComposer
    extends Composer<_$MaintenanceDatabase, $CachedExecutionsTable> {
  $$CachedExecutionsTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get id =>
      $composableBuilder(column: $table.id, builder: (column) => column);

  GeneratedColumn<String> get siteId =>
      $composableBuilder(column: $table.siteId, builder: (column) => column);

  GeneratedColumn<String> get incidentId => $composableBuilder(
    column: $table.incidentId,
    builder: (column) => column,
  );

  GeneratedColumn<String> get assetId =>
      $composableBuilder(column: $table.assetId, builder: (column) => column);

  GeneratedColumn<String> get assetTag =>
      $composableBuilder(column: $table.assetTag, builder: (column) => column);

  GeneratedColumn<String> get purpose =>
      $composableBuilder(column: $table.purpose, builder: (column) => column);

  GeneratedColumn<String> get state =>
      $composableBuilder(column: $table.state, builder: (column) => column);

  GeneratedColumn<int> get serverVersion => $composableBuilder(
    column: $table.serverVersion,
    builder: (column) => column,
  );

  GeneratedColumn<String> get verifiedPrerequisites => $composableBuilder(
    column: $table.verifiedPrerequisites,
    builder: (column) => column,
  );

  GeneratedColumn<DateTime> get cachedAt =>
      $composableBuilder(column: $table.cachedAt, builder: (column) => column);
}

class $$CachedExecutionsTableTableManager
    extends
        RootTableManager<
          _$MaintenanceDatabase,
          $CachedExecutionsTable,
          CachedExecution,
          $$CachedExecutionsTableFilterComposer,
          $$CachedExecutionsTableOrderingComposer,
          $$CachedExecutionsTableAnnotationComposer,
          $$CachedExecutionsTableCreateCompanionBuilder,
          $$CachedExecutionsTableUpdateCompanionBuilder,
          (
            CachedExecution,
            BaseReferences<
              _$MaintenanceDatabase,
              $CachedExecutionsTable,
              CachedExecution
            >,
          ),
          CachedExecution,
          PrefetchHooks Function()
        > {
  $$CachedExecutionsTableTableManager(
    _$MaintenanceDatabase db,
    $CachedExecutionsTable table,
  ) : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$CachedExecutionsTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$CachedExecutionsTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$CachedExecutionsTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> id = const Value.absent(),
                Value<String> siteId = const Value.absent(),
                Value<String> incidentId = const Value.absent(),
                Value<String> assetId = const Value.absent(),
                Value<String> assetTag = const Value.absent(),
                Value<String> purpose = const Value.absent(),
                Value<String> state = const Value.absent(),
                Value<int> serverVersion = const Value.absent(),
                Value<String> verifiedPrerequisites = const Value.absent(),
                Value<DateTime> cachedAt = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => CachedExecutionsCompanion(
                id: id,
                siteId: siteId,
                incidentId: incidentId,
                assetId: assetId,
                assetTag: assetTag,
                purpose: purpose,
                state: state,
                serverVersion: serverVersion,
                verifiedPrerequisites: verifiedPrerequisites,
                cachedAt: cachedAt,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String id,
                required String siteId,
                required String incidentId,
                required String assetId,
                required String assetTag,
                required String purpose,
                required String state,
                required int serverVersion,
                Value<String> verifiedPrerequisites = const Value.absent(),
                required DateTime cachedAt,
                Value<int> rowid = const Value.absent(),
              }) => CachedExecutionsCompanion.insert(
                id: id,
                siteId: siteId,
                incidentId: incidentId,
                assetId: assetId,
                assetTag: assetTag,
                purpose: purpose,
                state: state,
                serverVersion: serverVersion,
                verifiedPrerequisites: verifiedPrerequisites,
                cachedAt: cachedAt,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$CachedExecutionsTableProcessedTableManager =
    ProcessedTableManager<
      _$MaintenanceDatabase,
      $CachedExecutionsTable,
      CachedExecution,
      $$CachedExecutionsTableFilterComposer,
      $$CachedExecutionsTableOrderingComposer,
      $$CachedExecutionsTableAnnotationComposer,
      $$CachedExecutionsTableCreateCompanionBuilder,
      $$CachedExecutionsTableUpdateCompanionBuilder,
      (
        CachedExecution,
        BaseReferences<
          _$MaintenanceDatabase,
          $CachedExecutionsTable,
          CachedExecution
        >,
      ),
      CachedExecution,
      PrefetchHooks Function()
    >;
typedef $$CachedExecutionStepsTableCreateCompanionBuilder =
    CachedExecutionStepsCompanion Function({
      required String id,
      required String executionId,
      required String stepKey,
      required int sequence,
      required String title,
      required String state,
      required String riskLevel,
      Value<String?> requiredPrerequisite,
      Value<String?> blockedReason,
      required int serverVersion,
      Value<int> rowid,
    });
typedef $$CachedExecutionStepsTableUpdateCompanionBuilder =
    CachedExecutionStepsCompanion Function({
      Value<String> id,
      Value<String> executionId,
      Value<String> stepKey,
      Value<int> sequence,
      Value<String> title,
      Value<String> state,
      Value<String> riskLevel,
      Value<String?> requiredPrerequisite,
      Value<String?> blockedReason,
      Value<int> serverVersion,
      Value<int> rowid,
    });

class $$CachedExecutionStepsTableFilterComposer
    extends Composer<_$MaintenanceDatabase, $CachedExecutionStepsTable> {
  $$CachedExecutionStepsTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get executionId => $composableBuilder(
    column: $table.executionId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get stepKey => $composableBuilder(
    column: $table.stepKey,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get sequence => $composableBuilder(
    column: $table.sequence,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get title => $composableBuilder(
    column: $table.title,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get state => $composableBuilder(
    column: $table.state,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get riskLevel => $composableBuilder(
    column: $table.riskLevel,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get requiredPrerequisite => $composableBuilder(
    column: $table.requiredPrerequisite,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get blockedReason => $composableBuilder(
    column: $table.blockedReason,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get serverVersion => $composableBuilder(
    column: $table.serverVersion,
    builder: (column) => ColumnFilters(column),
  );
}

class $$CachedExecutionStepsTableOrderingComposer
    extends Composer<_$MaintenanceDatabase, $CachedExecutionStepsTable> {
  $$CachedExecutionStepsTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get executionId => $composableBuilder(
    column: $table.executionId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get stepKey => $composableBuilder(
    column: $table.stepKey,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get sequence => $composableBuilder(
    column: $table.sequence,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get title => $composableBuilder(
    column: $table.title,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get state => $composableBuilder(
    column: $table.state,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get riskLevel => $composableBuilder(
    column: $table.riskLevel,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get requiredPrerequisite => $composableBuilder(
    column: $table.requiredPrerequisite,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get blockedReason => $composableBuilder(
    column: $table.blockedReason,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get serverVersion => $composableBuilder(
    column: $table.serverVersion,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$CachedExecutionStepsTableAnnotationComposer
    extends Composer<_$MaintenanceDatabase, $CachedExecutionStepsTable> {
  $$CachedExecutionStepsTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get id =>
      $composableBuilder(column: $table.id, builder: (column) => column);

  GeneratedColumn<String> get executionId => $composableBuilder(
    column: $table.executionId,
    builder: (column) => column,
  );

  GeneratedColumn<String> get stepKey =>
      $composableBuilder(column: $table.stepKey, builder: (column) => column);

  GeneratedColumn<int> get sequence =>
      $composableBuilder(column: $table.sequence, builder: (column) => column);

  GeneratedColumn<String> get title =>
      $composableBuilder(column: $table.title, builder: (column) => column);

  GeneratedColumn<String> get state =>
      $composableBuilder(column: $table.state, builder: (column) => column);

  GeneratedColumn<String> get riskLevel =>
      $composableBuilder(column: $table.riskLevel, builder: (column) => column);

  GeneratedColumn<String> get requiredPrerequisite => $composableBuilder(
    column: $table.requiredPrerequisite,
    builder: (column) => column,
  );

  GeneratedColumn<String> get blockedReason => $composableBuilder(
    column: $table.blockedReason,
    builder: (column) => column,
  );

  GeneratedColumn<int> get serverVersion => $composableBuilder(
    column: $table.serverVersion,
    builder: (column) => column,
  );
}

class $$CachedExecutionStepsTableTableManager
    extends
        RootTableManager<
          _$MaintenanceDatabase,
          $CachedExecutionStepsTable,
          CachedExecutionStep,
          $$CachedExecutionStepsTableFilterComposer,
          $$CachedExecutionStepsTableOrderingComposer,
          $$CachedExecutionStepsTableAnnotationComposer,
          $$CachedExecutionStepsTableCreateCompanionBuilder,
          $$CachedExecutionStepsTableUpdateCompanionBuilder,
          (
            CachedExecutionStep,
            BaseReferences<
              _$MaintenanceDatabase,
              $CachedExecutionStepsTable,
              CachedExecutionStep
            >,
          ),
          CachedExecutionStep,
          PrefetchHooks Function()
        > {
  $$CachedExecutionStepsTableTableManager(
    _$MaintenanceDatabase db,
    $CachedExecutionStepsTable table,
  ) : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$CachedExecutionStepsTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$CachedExecutionStepsTableOrderingComposer(
                $db: db,
                $table: table,
              ),
          createComputedFieldComposer: () =>
              $$CachedExecutionStepsTableAnnotationComposer(
                $db: db,
                $table: table,
              ),
          updateCompanionCallback:
              ({
                Value<String> id = const Value.absent(),
                Value<String> executionId = const Value.absent(),
                Value<String> stepKey = const Value.absent(),
                Value<int> sequence = const Value.absent(),
                Value<String> title = const Value.absent(),
                Value<String> state = const Value.absent(),
                Value<String> riskLevel = const Value.absent(),
                Value<String?> requiredPrerequisite = const Value.absent(),
                Value<String?> blockedReason = const Value.absent(),
                Value<int> serverVersion = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => CachedExecutionStepsCompanion(
                id: id,
                executionId: executionId,
                stepKey: stepKey,
                sequence: sequence,
                title: title,
                state: state,
                riskLevel: riskLevel,
                requiredPrerequisite: requiredPrerequisite,
                blockedReason: blockedReason,
                serverVersion: serverVersion,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String id,
                required String executionId,
                required String stepKey,
                required int sequence,
                required String title,
                required String state,
                required String riskLevel,
                Value<String?> requiredPrerequisite = const Value.absent(),
                Value<String?> blockedReason = const Value.absent(),
                required int serverVersion,
                Value<int> rowid = const Value.absent(),
              }) => CachedExecutionStepsCompanion.insert(
                id: id,
                executionId: executionId,
                stepKey: stepKey,
                sequence: sequence,
                title: title,
                state: state,
                riskLevel: riskLevel,
                requiredPrerequisite: requiredPrerequisite,
                blockedReason: blockedReason,
                serverVersion: serverVersion,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$CachedExecutionStepsTableProcessedTableManager =
    ProcessedTableManager<
      _$MaintenanceDatabase,
      $CachedExecutionStepsTable,
      CachedExecutionStep,
      $$CachedExecutionStepsTableFilterComposer,
      $$CachedExecutionStepsTableOrderingComposer,
      $$CachedExecutionStepsTableAnnotationComposer,
      $$CachedExecutionStepsTableCreateCompanionBuilder,
      $$CachedExecutionStepsTableUpdateCompanionBuilder,
      (
        CachedExecutionStep,
        BaseReferences<
          _$MaintenanceDatabase,
          $CachedExecutionStepsTable,
          CachedExecutionStep
        >,
      ),
      CachedExecutionStep,
      PrefetchHooks Function()
    >;
typedef $$LocalMeasurementsTableCreateCompanionBuilder =
    LocalMeasurementsCompanion Function({
      required String clientEventId,
      required String executionId,
      Value<String?> componentId,
      required String measurementType,
      required String value,
      required String unit,
      Value<String> source,
      Value<String> dataQuality,
      Value<String> verificationStatus,
      required DateTime observedAt,
      required DateTime createdAtDevice,
      Value<String> syncStatus,
      Value<String?> serverId,
      Value<String?> syncError,
      Value<int> rowid,
    });
typedef $$LocalMeasurementsTableUpdateCompanionBuilder =
    LocalMeasurementsCompanion Function({
      Value<String> clientEventId,
      Value<String> executionId,
      Value<String?> componentId,
      Value<String> measurementType,
      Value<String> value,
      Value<String> unit,
      Value<String> source,
      Value<String> dataQuality,
      Value<String> verificationStatus,
      Value<DateTime> observedAt,
      Value<DateTime> createdAtDevice,
      Value<String> syncStatus,
      Value<String?> serverId,
      Value<String?> syncError,
      Value<int> rowid,
    });

class $$LocalMeasurementsTableFilterComposer
    extends Composer<_$MaintenanceDatabase, $LocalMeasurementsTable> {
  $$LocalMeasurementsTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get clientEventId => $composableBuilder(
    column: $table.clientEventId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get executionId => $composableBuilder(
    column: $table.executionId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get componentId => $composableBuilder(
    column: $table.componentId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get measurementType => $composableBuilder(
    column: $table.measurementType,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get value => $composableBuilder(
    column: $table.value,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get unit => $composableBuilder(
    column: $table.unit,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get source => $composableBuilder(
    column: $table.source,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get dataQuality => $composableBuilder(
    column: $table.dataQuality,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get verificationStatus => $composableBuilder(
    column: $table.verificationStatus,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get observedAt => $composableBuilder(
    column: $table.observedAt,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get createdAtDevice => $composableBuilder(
    column: $table.createdAtDevice,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get syncStatus => $composableBuilder(
    column: $table.syncStatus,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get serverId => $composableBuilder(
    column: $table.serverId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get syncError => $composableBuilder(
    column: $table.syncError,
    builder: (column) => ColumnFilters(column),
  );
}

class $$LocalMeasurementsTableOrderingComposer
    extends Composer<_$MaintenanceDatabase, $LocalMeasurementsTable> {
  $$LocalMeasurementsTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get clientEventId => $composableBuilder(
    column: $table.clientEventId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get executionId => $composableBuilder(
    column: $table.executionId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get componentId => $composableBuilder(
    column: $table.componentId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get measurementType => $composableBuilder(
    column: $table.measurementType,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get value => $composableBuilder(
    column: $table.value,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get unit => $composableBuilder(
    column: $table.unit,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get source => $composableBuilder(
    column: $table.source,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get dataQuality => $composableBuilder(
    column: $table.dataQuality,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get verificationStatus => $composableBuilder(
    column: $table.verificationStatus,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get observedAt => $composableBuilder(
    column: $table.observedAt,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get createdAtDevice => $composableBuilder(
    column: $table.createdAtDevice,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get syncStatus => $composableBuilder(
    column: $table.syncStatus,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get serverId => $composableBuilder(
    column: $table.serverId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get syncError => $composableBuilder(
    column: $table.syncError,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$LocalMeasurementsTableAnnotationComposer
    extends Composer<_$MaintenanceDatabase, $LocalMeasurementsTable> {
  $$LocalMeasurementsTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get clientEventId => $composableBuilder(
    column: $table.clientEventId,
    builder: (column) => column,
  );

  GeneratedColumn<String> get executionId => $composableBuilder(
    column: $table.executionId,
    builder: (column) => column,
  );

  GeneratedColumn<String> get componentId => $composableBuilder(
    column: $table.componentId,
    builder: (column) => column,
  );

  GeneratedColumn<String> get measurementType => $composableBuilder(
    column: $table.measurementType,
    builder: (column) => column,
  );

  GeneratedColumn<String> get value =>
      $composableBuilder(column: $table.value, builder: (column) => column);

  GeneratedColumn<String> get unit =>
      $composableBuilder(column: $table.unit, builder: (column) => column);

  GeneratedColumn<String> get source =>
      $composableBuilder(column: $table.source, builder: (column) => column);

  GeneratedColumn<String> get dataQuality => $composableBuilder(
    column: $table.dataQuality,
    builder: (column) => column,
  );

  GeneratedColumn<String> get verificationStatus => $composableBuilder(
    column: $table.verificationStatus,
    builder: (column) => column,
  );

  GeneratedColumn<DateTime> get observedAt => $composableBuilder(
    column: $table.observedAt,
    builder: (column) => column,
  );

  GeneratedColumn<DateTime> get createdAtDevice => $composableBuilder(
    column: $table.createdAtDevice,
    builder: (column) => column,
  );

  GeneratedColumn<String> get syncStatus => $composableBuilder(
    column: $table.syncStatus,
    builder: (column) => column,
  );

  GeneratedColumn<String> get serverId =>
      $composableBuilder(column: $table.serverId, builder: (column) => column);

  GeneratedColumn<String> get syncError =>
      $composableBuilder(column: $table.syncError, builder: (column) => column);
}

class $$LocalMeasurementsTableTableManager
    extends
        RootTableManager<
          _$MaintenanceDatabase,
          $LocalMeasurementsTable,
          LocalMeasurement,
          $$LocalMeasurementsTableFilterComposer,
          $$LocalMeasurementsTableOrderingComposer,
          $$LocalMeasurementsTableAnnotationComposer,
          $$LocalMeasurementsTableCreateCompanionBuilder,
          $$LocalMeasurementsTableUpdateCompanionBuilder,
          (
            LocalMeasurement,
            BaseReferences<
              _$MaintenanceDatabase,
              $LocalMeasurementsTable,
              LocalMeasurement
            >,
          ),
          LocalMeasurement,
          PrefetchHooks Function()
        > {
  $$LocalMeasurementsTableTableManager(
    _$MaintenanceDatabase db,
    $LocalMeasurementsTable table,
  ) : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$LocalMeasurementsTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$LocalMeasurementsTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$LocalMeasurementsTableAnnotationComposer(
                $db: db,
                $table: table,
              ),
          updateCompanionCallback:
              ({
                Value<String> clientEventId = const Value.absent(),
                Value<String> executionId = const Value.absent(),
                Value<String?> componentId = const Value.absent(),
                Value<String> measurementType = const Value.absent(),
                Value<String> value = const Value.absent(),
                Value<String> unit = const Value.absent(),
                Value<String> source = const Value.absent(),
                Value<String> dataQuality = const Value.absent(),
                Value<String> verificationStatus = const Value.absent(),
                Value<DateTime> observedAt = const Value.absent(),
                Value<DateTime> createdAtDevice = const Value.absent(),
                Value<String> syncStatus = const Value.absent(),
                Value<String?> serverId = const Value.absent(),
                Value<String?> syncError = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => LocalMeasurementsCompanion(
                clientEventId: clientEventId,
                executionId: executionId,
                componentId: componentId,
                measurementType: measurementType,
                value: value,
                unit: unit,
                source: source,
                dataQuality: dataQuality,
                verificationStatus: verificationStatus,
                observedAt: observedAt,
                createdAtDevice: createdAtDevice,
                syncStatus: syncStatus,
                serverId: serverId,
                syncError: syncError,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String clientEventId,
                required String executionId,
                Value<String?> componentId = const Value.absent(),
                required String measurementType,
                required String value,
                required String unit,
                Value<String> source = const Value.absent(),
                Value<String> dataQuality = const Value.absent(),
                Value<String> verificationStatus = const Value.absent(),
                required DateTime observedAt,
                required DateTime createdAtDevice,
                Value<String> syncStatus = const Value.absent(),
                Value<String?> serverId = const Value.absent(),
                Value<String?> syncError = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => LocalMeasurementsCompanion.insert(
                clientEventId: clientEventId,
                executionId: executionId,
                componentId: componentId,
                measurementType: measurementType,
                value: value,
                unit: unit,
                source: source,
                dataQuality: dataQuality,
                verificationStatus: verificationStatus,
                observedAt: observedAt,
                createdAtDevice: createdAtDevice,
                syncStatus: syncStatus,
                serverId: serverId,
                syncError: syncError,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$LocalMeasurementsTableProcessedTableManager =
    ProcessedTableManager<
      _$MaintenanceDatabase,
      $LocalMeasurementsTable,
      LocalMeasurement,
      $$LocalMeasurementsTableFilterComposer,
      $$LocalMeasurementsTableOrderingComposer,
      $$LocalMeasurementsTableAnnotationComposer,
      $$LocalMeasurementsTableCreateCompanionBuilder,
      $$LocalMeasurementsTableUpdateCompanionBuilder,
      (
        LocalMeasurement,
        BaseReferences<
          _$MaintenanceDatabase,
          $LocalMeasurementsTable,
          LocalMeasurement
        >,
      ),
      LocalMeasurement,
      PrefetchHooks Function()
    >;
typedef $$LocalObservationsTableCreateCompanionBuilder =
    LocalObservationsCompanion Function({
      required String clientEventId,
      required String executionId,
      Value<String?> componentId,
      Value<String?> property,
      Value<String?> status,
      required String narrative,
      Value<String> source,
      Value<String> verificationStatus,
      required DateTime observedAt,
      required DateTime createdAtDevice,
      Value<String> syncStatus,
      Value<String?> serverId,
      Value<String?> syncError,
      Value<int> rowid,
    });
typedef $$LocalObservationsTableUpdateCompanionBuilder =
    LocalObservationsCompanion Function({
      Value<String> clientEventId,
      Value<String> executionId,
      Value<String?> componentId,
      Value<String?> property,
      Value<String?> status,
      Value<String> narrative,
      Value<String> source,
      Value<String> verificationStatus,
      Value<DateTime> observedAt,
      Value<DateTime> createdAtDevice,
      Value<String> syncStatus,
      Value<String?> serverId,
      Value<String?> syncError,
      Value<int> rowid,
    });

class $$LocalObservationsTableFilterComposer
    extends Composer<_$MaintenanceDatabase, $LocalObservationsTable> {
  $$LocalObservationsTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get clientEventId => $composableBuilder(
    column: $table.clientEventId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get executionId => $composableBuilder(
    column: $table.executionId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get componentId => $composableBuilder(
    column: $table.componentId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get property => $composableBuilder(
    column: $table.property,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get status => $composableBuilder(
    column: $table.status,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get narrative => $composableBuilder(
    column: $table.narrative,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get source => $composableBuilder(
    column: $table.source,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get verificationStatus => $composableBuilder(
    column: $table.verificationStatus,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get observedAt => $composableBuilder(
    column: $table.observedAt,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get createdAtDevice => $composableBuilder(
    column: $table.createdAtDevice,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get syncStatus => $composableBuilder(
    column: $table.syncStatus,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get serverId => $composableBuilder(
    column: $table.serverId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get syncError => $composableBuilder(
    column: $table.syncError,
    builder: (column) => ColumnFilters(column),
  );
}

class $$LocalObservationsTableOrderingComposer
    extends Composer<_$MaintenanceDatabase, $LocalObservationsTable> {
  $$LocalObservationsTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get clientEventId => $composableBuilder(
    column: $table.clientEventId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get executionId => $composableBuilder(
    column: $table.executionId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get componentId => $composableBuilder(
    column: $table.componentId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get property => $composableBuilder(
    column: $table.property,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get status => $composableBuilder(
    column: $table.status,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get narrative => $composableBuilder(
    column: $table.narrative,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get source => $composableBuilder(
    column: $table.source,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get verificationStatus => $composableBuilder(
    column: $table.verificationStatus,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get observedAt => $composableBuilder(
    column: $table.observedAt,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get createdAtDevice => $composableBuilder(
    column: $table.createdAtDevice,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get syncStatus => $composableBuilder(
    column: $table.syncStatus,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get serverId => $composableBuilder(
    column: $table.serverId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get syncError => $composableBuilder(
    column: $table.syncError,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$LocalObservationsTableAnnotationComposer
    extends Composer<_$MaintenanceDatabase, $LocalObservationsTable> {
  $$LocalObservationsTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get clientEventId => $composableBuilder(
    column: $table.clientEventId,
    builder: (column) => column,
  );

  GeneratedColumn<String> get executionId => $composableBuilder(
    column: $table.executionId,
    builder: (column) => column,
  );

  GeneratedColumn<String> get componentId => $composableBuilder(
    column: $table.componentId,
    builder: (column) => column,
  );

  GeneratedColumn<String> get property =>
      $composableBuilder(column: $table.property, builder: (column) => column);

  GeneratedColumn<String> get status =>
      $composableBuilder(column: $table.status, builder: (column) => column);

  GeneratedColumn<String> get narrative =>
      $composableBuilder(column: $table.narrative, builder: (column) => column);

  GeneratedColumn<String> get source =>
      $composableBuilder(column: $table.source, builder: (column) => column);

  GeneratedColumn<String> get verificationStatus => $composableBuilder(
    column: $table.verificationStatus,
    builder: (column) => column,
  );

  GeneratedColumn<DateTime> get observedAt => $composableBuilder(
    column: $table.observedAt,
    builder: (column) => column,
  );

  GeneratedColumn<DateTime> get createdAtDevice => $composableBuilder(
    column: $table.createdAtDevice,
    builder: (column) => column,
  );

  GeneratedColumn<String> get syncStatus => $composableBuilder(
    column: $table.syncStatus,
    builder: (column) => column,
  );

  GeneratedColumn<String> get serverId =>
      $composableBuilder(column: $table.serverId, builder: (column) => column);

  GeneratedColumn<String> get syncError =>
      $composableBuilder(column: $table.syncError, builder: (column) => column);
}

class $$LocalObservationsTableTableManager
    extends
        RootTableManager<
          _$MaintenanceDatabase,
          $LocalObservationsTable,
          LocalObservation,
          $$LocalObservationsTableFilterComposer,
          $$LocalObservationsTableOrderingComposer,
          $$LocalObservationsTableAnnotationComposer,
          $$LocalObservationsTableCreateCompanionBuilder,
          $$LocalObservationsTableUpdateCompanionBuilder,
          (
            LocalObservation,
            BaseReferences<
              _$MaintenanceDatabase,
              $LocalObservationsTable,
              LocalObservation
            >,
          ),
          LocalObservation,
          PrefetchHooks Function()
        > {
  $$LocalObservationsTableTableManager(
    _$MaintenanceDatabase db,
    $LocalObservationsTable table,
  ) : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$LocalObservationsTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$LocalObservationsTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$LocalObservationsTableAnnotationComposer(
                $db: db,
                $table: table,
              ),
          updateCompanionCallback:
              ({
                Value<String> clientEventId = const Value.absent(),
                Value<String> executionId = const Value.absent(),
                Value<String?> componentId = const Value.absent(),
                Value<String?> property = const Value.absent(),
                Value<String?> status = const Value.absent(),
                Value<String> narrative = const Value.absent(),
                Value<String> source = const Value.absent(),
                Value<String> verificationStatus = const Value.absent(),
                Value<DateTime> observedAt = const Value.absent(),
                Value<DateTime> createdAtDevice = const Value.absent(),
                Value<String> syncStatus = const Value.absent(),
                Value<String?> serverId = const Value.absent(),
                Value<String?> syncError = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => LocalObservationsCompanion(
                clientEventId: clientEventId,
                executionId: executionId,
                componentId: componentId,
                property: property,
                status: status,
                narrative: narrative,
                source: source,
                verificationStatus: verificationStatus,
                observedAt: observedAt,
                createdAtDevice: createdAtDevice,
                syncStatus: syncStatus,
                serverId: serverId,
                syncError: syncError,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String clientEventId,
                required String executionId,
                Value<String?> componentId = const Value.absent(),
                Value<String?> property = const Value.absent(),
                Value<String?> status = const Value.absent(),
                required String narrative,
                Value<String> source = const Value.absent(),
                Value<String> verificationStatus = const Value.absent(),
                required DateTime observedAt,
                required DateTime createdAtDevice,
                Value<String> syncStatus = const Value.absent(),
                Value<String?> serverId = const Value.absent(),
                Value<String?> syncError = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => LocalObservationsCompanion.insert(
                clientEventId: clientEventId,
                executionId: executionId,
                componentId: componentId,
                property: property,
                status: status,
                narrative: narrative,
                source: source,
                verificationStatus: verificationStatus,
                observedAt: observedAt,
                createdAtDevice: createdAtDevice,
                syncStatus: syncStatus,
                serverId: serverId,
                syncError: syncError,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$LocalObservationsTableProcessedTableManager =
    ProcessedTableManager<
      _$MaintenanceDatabase,
      $LocalObservationsTable,
      LocalObservation,
      $$LocalObservationsTableFilterComposer,
      $$LocalObservationsTableOrderingComposer,
      $$LocalObservationsTableAnnotationComposer,
      $$LocalObservationsTableCreateCompanionBuilder,
      $$LocalObservationsTableUpdateCompanionBuilder,
      (
        LocalObservation,
        BaseReferences<
          _$MaintenanceDatabase,
          $LocalObservationsTable,
          LocalObservation
        >,
      ),
      LocalObservation,
      PrefetchHooks Function()
    >;
typedef $$LocalAttachmentsTableCreateCompanionBuilder =
    LocalAttachmentsCompanion Function({
      required String clientEventId,
      required String entityKind,
      required String entityId,
      required String siteId,
      required String localPath,
      required String filename,
      required String mimeType,
      required int sizeBytes,
      required String checksumSha256,
      Value<String> syncStatus,
      Value<String?> serverId,
      Value<String?> syncError,
      required DateTime createdAtDevice,
      Value<int> rowid,
    });
typedef $$LocalAttachmentsTableUpdateCompanionBuilder =
    LocalAttachmentsCompanion Function({
      Value<String> clientEventId,
      Value<String> entityKind,
      Value<String> entityId,
      Value<String> siteId,
      Value<String> localPath,
      Value<String> filename,
      Value<String> mimeType,
      Value<int> sizeBytes,
      Value<String> checksumSha256,
      Value<String> syncStatus,
      Value<String?> serverId,
      Value<String?> syncError,
      Value<DateTime> createdAtDevice,
      Value<int> rowid,
    });

class $$LocalAttachmentsTableFilterComposer
    extends Composer<_$MaintenanceDatabase, $LocalAttachmentsTable> {
  $$LocalAttachmentsTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get clientEventId => $composableBuilder(
    column: $table.clientEventId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get entityKind => $composableBuilder(
    column: $table.entityKind,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get entityId => $composableBuilder(
    column: $table.entityId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get siteId => $composableBuilder(
    column: $table.siteId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get localPath => $composableBuilder(
    column: $table.localPath,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get filename => $composableBuilder(
    column: $table.filename,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get mimeType => $composableBuilder(
    column: $table.mimeType,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get sizeBytes => $composableBuilder(
    column: $table.sizeBytes,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get checksumSha256 => $composableBuilder(
    column: $table.checksumSha256,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get syncStatus => $composableBuilder(
    column: $table.syncStatus,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get serverId => $composableBuilder(
    column: $table.serverId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get syncError => $composableBuilder(
    column: $table.syncError,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get createdAtDevice => $composableBuilder(
    column: $table.createdAtDevice,
    builder: (column) => ColumnFilters(column),
  );
}

class $$LocalAttachmentsTableOrderingComposer
    extends Composer<_$MaintenanceDatabase, $LocalAttachmentsTable> {
  $$LocalAttachmentsTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get clientEventId => $composableBuilder(
    column: $table.clientEventId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get entityKind => $composableBuilder(
    column: $table.entityKind,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get entityId => $composableBuilder(
    column: $table.entityId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get siteId => $composableBuilder(
    column: $table.siteId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get localPath => $composableBuilder(
    column: $table.localPath,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get filename => $composableBuilder(
    column: $table.filename,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get mimeType => $composableBuilder(
    column: $table.mimeType,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get sizeBytes => $composableBuilder(
    column: $table.sizeBytes,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get checksumSha256 => $composableBuilder(
    column: $table.checksumSha256,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get syncStatus => $composableBuilder(
    column: $table.syncStatus,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get serverId => $composableBuilder(
    column: $table.serverId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get syncError => $composableBuilder(
    column: $table.syncError,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get createdAtDevice => $composableBuilder(
    column: $table.createdAtDevice,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$LocalAttachmentsTableAnnotationComposer
    extends Composer<_$MaintenanceDatabase, $LocalAttachmentsTable> {
  $$LocalAttachmentsTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get clientEventId => $composableBuilder(
    column: $table.clientEventId,
    builder: (column) => column,
  );

  GeneratedColumn<String> get entityKind => $composableBuilder(
    column: $table.entityKind,
    builder: (column) => column,
  );

  GeneratedColumn<String> get entityId =>
      $composableBuilder(column: $table.entityId, builder: (column) => column);

  GeneratedColumn<String> get siteId =>
      $composableBuilder(column: $table.siteId, builder: (column) => column);

  GeneratedColumn<String> get localPath =>
      $composableBuilder(column: $table.localPath, builder: (column) => column);

  GeneratedColumn<String> get filename =>
      $composableBuilder(column: $table.filename, builder: (column) => column);

  GeneratedColumn<String> get mimeType =>
      $composableBuilder(column: $table.mimeType, builder: (column) => column);

  GeneratedColumn<int> get sizeBytes =>
      $composableBuilder(column: $table.sizeBytes, builder: (column) => column);

  GeneratedColumn<String> get checksumSha256 => $composableBuilder(
    column: $table.checksumSha256,
    builder: (column) => column,
  );

  GeneratedColumn<String> get syncStatus => $composableBuilder(
    column: $table.syncStatus,
    builder: (column) => column,
  );

  GeneratedColumn<String> get serverId =>
      $composableBuilder(column: $table.serverId, builder: (column) => column);

  GeneratedColumn<String> get syncError =>
      $composableBuilder(column: $table.syncError, builder: (column) => column);

  GeneratedColumn<DateTime> get createdAtDevice => $composableBuilder(
    column: $table.createdAtDevice,
    builder: (column) => column,
  );
}

class $$LocalAttachmentsTableTableManager
    extends
        RootTableManager<
          _$MaintenanceDatabase,
          $LocalAttachmentsTable,
          LocalAttachment,
          $$LocalAttachmentsTableFilterComposer,
          $$LocalAttachmentsTableOrderingComposer,
          $$LocalAttachmentsTableAnnotationComposer,
          $$LocalAttachmentsTableCreateCompanionBuilder,
          $$LocalAttachmentsTableUpdateCompanionBuilder,
          (
            LocalAttachment,
            BaseReferences<
              _$MaintenanceDatabase,
              $LocalAttachmentsTable,
              LocalAttachment
            >,
          ),
          LocalAttachment,
          PrefetchHooks Function()
        > {
  $$LocalAttachmentsTableTableManager(
    _$MaintenanceDatabase db,
    $LocalAttachmentsTable table,
  ) : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$LocalAttachmentsTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$LocalAttachmentsTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$LocalAttachmentsTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> clientEventId = const Value.absent(),
                Value<String> entityKind = const Value.absent(),
                Value<String> entityId = const Value.absent(),
                Value<String> siteId = const Value.absent(),
                Value<String> localPath = const Value.absent(),
                Value<String> filename = const Value.absent(),
                Value<String> mimeType = const Value.absent(),
                Value<int> sizeBytes = const Value.absent(),
                Value<String> checksumSha256 = const Value.absent(),
                Value<String> syncStatus = const Value.absent(),
                Value<String?> serverId = const Value.absent(),
                Value<String?> syncError = const Value.absent(),
                Value<DateTime> createdAtDevice = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => LocalAttachmentsCompanion(
                clientEventId: clientEventId,
                entityKind: entityKind,
                entityId: entityId,
                siteId: siteId,
                localPath: localPath,
                filename: filename,
                mimeType: mimeType,
                sizeBytes: sizeBytes,
                checksumSha256: checksumSha256,
                syncStatus: syncStatus,
                serverId: serverId,
                syncError: syncError,
                createdAtDevice: createdAtDevice,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String clientEventId,
                required String entityKind,
                required String entityId,
                required String siteId,
                required String localPath,
                required String filename,
                required String mimeType,
                required int sizeBytes,
                required String checksumSha256,
                Value<String> syncStatus = const Value.absent(),
                Value<String?> serverId = const Value.absent(),
                Value<String?> syncError = const Value.absent(),
                required DateTime createdAtDevice,
                Value<int> rowid = const Value.absent(),
              }) => LocalAttachmentsCompanion.insert(
                clientEventId: clientEventId,
                entityKind: entityKind,
                entityId: entityId,
                siteId: siteId,
                localPath: localPath,
                filename: filename,
                mimeType: mimeType,
                sizeBytes: sizeBytes,
                checksumSha256: checksumSha256,
                syncStatus: syncStatus,
                serverId: serverId,
                syncError: syncError,
                createdAtDevice: createdAtDevice,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$LocalAttachmentsTableProcessedTableManager =
    ProcessedTableManager<
      _$MaintenanceDatabase,
      $LocalAttachmentsTable,
      LocalAttachment,
      $$LocalAttachmentsTableFilterComposer,
      $$LocalAttachmentsTableOrderingComposer,
      $$LocalAttachmentsTableAnnotationComposer,
      $$LocalAttachmentsTableCreateCompanionBuilder,
      $$LocalAttachmentsTableUpdateCompanionBuilder,
      (
        LocalAttachment,
        BaseReferences<
          _$MaintenanceDatabase,
          $LocalAttachmentsTable,
          LocalAttachment
        >,
      ),
      LocalAttachment,
      PrefetchHooks Function()
    >;
typedef $$OutboxEntriesTableCreateCompanionBuilder =
    OutboxEntriesCompanion Function({
      required String id,
      required String clientEventId,
      required String idempotencyKey,
      required String operation,
      required String path,
      required String payloadJson,
      Value<String> status,
      Value<int> attemptCount,
      required DateTime nextAttemptAt,
      required DateTime createdAtDevice,
      Value<DateTime?> lastAttemptAt,
      Value<String?> lastError,
      Value<int?> baseServerVersion,
      Value<int> rowid,
    });
typedef $$OutboxEntriesTableUpdateCompanionBuilder =
    OutboxEntriesCompanion Function({
      Value<String> id,
      Value<String> clientEventId,
      Value<String> idempotencyKey,
      Value<String> operation,
      Value<String> path,
      Value<String> payloadJson,
      Value<String> status,
      Value<int> attemptCount,
      Value<DateTime> nextAttemptAt,
      Value<DateTime> createdAtDevice,
      Value<DateTime?> lastAttemptAt,
      Value<String?> lastError,
      Value<int?> baseServerVersion,
      Value<int> rowid,
    });

class $$OutboxEntriesTableFilterComposer
    extends Composer<_$MaintenanceDatabase, $OutboxEntriesTable> {
  $$OutboxEntriesTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get clientEventId => $composableBuilder(
    column: $table.clientEventId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get idempotencyKey => $composableBuilder(
    column: $table.idempotencyKey,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get operation => $composableBuilder(
    column: $table.operation,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get path => $composableBuilder(
    column: $table.path,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get payloadJson => $composableBuilder(
    column: $table.payloadJson,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get status => $composableBuilder(
    column: $table.status,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get attemptCount => $composableBuilder(
    column: $table.attemptCount,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get nextAttemptAt => $composableBuilder(
    column: $table.nextAttemptAt,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get createdAtDevice => $composableBuilder(
    column: $table.createdAtDevice,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get lastAttemptAt => $composableBuilder(
    column: $table.lastAttemptAt,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get lastError => $composableBuilder(
    column: $table.lastError,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get baseServerVersion => $composableBuilder(
    column: $table.baseServerVersion,
    builder: (column) => ColumnFilters(column),
  );
}

class $$OutboxEntriesTableOrderingComposer
    extends Composer<_$MaintenanceDatabase, $OutboxEntriesTable> {
  $$OutboxEntriesTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get clientEventId => $composableBuilder(
    column: $table.clientEventId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get idempotencyKey => $composableBuilder(
    column: $table.idempotencyKey,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get operation => $composableBuilder(
    column: $table.operation,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get path => $composableBuilder(
    column: $table.path,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get payloadJson => $composableBuilder(
    column: $table.payloadJson,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get status => $composableBuilder(
    column: $table.status,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get attemptCount => $composableBuilder(
    column: $table.attemptCount,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get nextAttemptAt => $composableBuilder(
    column: $table.nextAttemptAt,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get createdAtDevice => $composableBuilder(
    column: $table.createdAtDevice,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get lastAttemptAt => $composableBuilder(
    column: $table.lastAttemptAt,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get lastError => $composableBuilder(
    column: $table.lastError,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get baseServerVersion => $composableBuilder(
    column: $table.baseServerVersion,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$OutboxEntriesTableAnnotationComposer
    extends Composer<_$MaintenanceDatabase, $OutboxEntriesTable> {
  $$OutboxEntriesTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get id =>
      $composableBuilder(column: $table.id, builder: (column) => column);

  GeneratedColumn<String> get clientEventId => $composableBuilder(
    column: $table.clientEventId,
    builder: (column) => column,
  );

  GeneratedColumn<String> get idempotencyKey => $composableBuilder(
    column: $table.idempotencyKey,
    builder: (column) => column,
  );

  GeneratedColumn<String> get operation =>
      $composableBuilder(column: $table.operation, builder: (column) => column);

  GeneratedColumn<String> get path =>
      $composableBuilder(column: $table.path, builder: (column) => column);

  GeneratedColumn<String> get payloadJson => $composableBuilder(
    column: $table.payloadJson,
    builder: (column) => column,
  );

  GeneratedColumn<String> get status =>
      $composableBuilder(column: $table.status, builder: (column) => column);

  GeneratedColumn<int> get attemptCount => $composableBuilder(
    column: $table.attemptCount,
    builder: (column) => column,
  );

  GeneratedColumn<DateTime> get nextAttemptAt => $composableBuilder(
    column: $table.nextAttemptAt,
    builder: (column) => column,
  );

  GeneratedColumn<DateTime> get createdAtDevice => $composableBuilder(
    column: $table.createdAtDevice,
    builder: (column) => column,
  );

  GeneratedColumn<DateTime> get lastAttemptAt => $composableBuilder(
    column: $table.lastAttemptAt,
    builder: (column) => column,
  );

  GeneratedColumn<String> get lastError =>
      $composableBuilder(column: $table.lastError, builder: (column) => column);

  GeneratedColumn<int> get baseServerVersion => $composableBuilder(
    column: $table.baseServerVersion,
    builder: (column) => column,
  );
}

class $$OutboxEntriesTableTableManager
    extends
        RootTableManager<
          _$MaintenanceDatabase,
          $OutboxEntriesTable,
          OutboxEntry,
          $$OutboxEntriesTableFilterComposer,
          $$OutboxEntriesTableOrderingComposer,
          $$OutboxEntriesTableAnnotationComposer,
          $$OutboxEntriesTableCreateCompanionBuilder,
          $$OutboxEntriesTableUpdateCompanionBuilder,
          (
            OutboxEntry,
            BaseReferences<
              _$MaintenanceDatabase,
              $OutboxEntriesTable,
              OutboxEntry
            >,
          ),
          OutboxEntry,
          PrefetchHooks Function()
        > {
  $$OutboxEntriesTableTableManager(
    _$MaintenanceDatabase db,
    $OutboxEntriesTable table,
  ) : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$OutboxEntriesTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$OutboxEntriesTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$OutboxEntriesTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> id = const Value.absent(),
                Value<String> clientEventId = const Value.absent(),
                Value<String> idempotencyKey = const Value.absent(),
                Value<String> operation = const Value.absent(),
                Value<String> path = const Value.absent(),
                Value<String> payloadJson = const Value.absent(),
                Value<String> status = const Value.absent(),
                Value<int> attemptCount = const Value.absent(),
                Value<DateTime> nextAttemptAt = const Value.absent(),
                Value<DateTime> createdAtDevice = const Value.absent(),
                Value<DateTime?> lastAttemptAt = const Value.absent(),
                Value<String?> lastError = const Value.absent(),
                Value<int?> baseServerVersion = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => OutboxEntriesCompanion(
                id: id,
                clientEventId: clientEventId,
                idempotencyKey: idempotencyKey,
                operation: operation,
                path: path,
                payloadJson: payloadJson,
                status: status,
                attemptCount: attemptCount,
                nextAttemptAt: nextAttemptAt,
                createdAtDevice: createdAtDevice,
                lastAttemptAt: lastAttemptAt,
                lastError: lastError,
                baseServerVersion: baseServerVersion,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String id,
                required String clientEventId,
                required String idempotencyKey,
                required String operation,
                required String path,
                required String payloadJson,
                Value<String> status = const Value.absent(),
                Value<int> attemptCount = const Value.absent(),
                required DateTime nextAttemptAt,
                required DateTime createdAtDevice,
                Value<DateTime?> lastAttemptAt = const Value.absent(),
                Value<String?> lastError = const Value.absent(),
                Value<int?> baseServerVersion = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => OutboxEntriesCompanion.insert(
                id: id,
                clientEventId: clientEventId,
                idempotencyKey: idempotencyKey,
                operation: operation,
                path: path,
                payloadJson: payloadJson,
                status: status,
                attemptCount: attemptCount,
                nextAttemptAt: nextAttemptAt,
                createdAtDevice: createdAtDevice,
                lastAttemptAt: lastAttemptAt,
                lastError: lastError,
                baseServerVersion: baseServerVersion,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$OutboxEntriesTableProcessedTableManager =
    ProcessedTableManager<
      _$MaintenanceDatabase,
      $OutboxEntriesTable,
      OutboxEntry,
      $$OutboxEntriesTableFilterComposer,
      $$OutboxEntriesTableOrderingComposer,
      $$OutboxEntriesTableAnnotationComposer,
      $$OutboxEntriesTableCreateCompanionBuilder,
      $$OutboxEntriesTableUpdateCompanionBuilder,
      (
        OutboxEntry,
        BaseReferences<_$MaintenanceDatabase, $OutboxEntriesTable, OutboxEntry>,
      ),
      OutboxEntry,
      PrefetchHooks Function()
    >;

class $MaintenanceDatabaseManager {
  final _$MaintenanceDatabase _db;
  $MaintenanceDatabaseManager(this._db);
  $$CachedAssetsTableTableManager get cachedAssets =>
      $$CachedAssetsTableTableManager(_db, _db.cachedAssets);
  $$CachedIncidentsTableTableManager get cachedIncidents =>
      $$CachedIncidentsTableTableManager(_db, _db.cachedIncidents);
  $$CachedExecutionsTableTableManager get cachedExecutions =>
      $$CachedExecutionsTableTableManager(_db, _db.cachedExecutions);
  $$CachedExecutionStepsTableTableManager get cachedExecutionSteps =>
      $$CachedExecutionStepsTableTableManager(_db, _db.cachedExecutionSteps);
  $$LocalMeasurementsTableTableManager get localMeasurements =>
      $$LocalMeasurementsTableTableManager(_db, _db.localMeasurements);
  $$LocalObservationsTableTableManager get localObservations =>
      $$LocalObservationsTableTableManager(_db, _db.localObservations);
  $$LocalAttachmentsTableTableManager get localAttachments =>
      $$LocalAttachmentsTableTableManager(_db, _db.localAttachments);
  $$OutboxEntriesTableTableManager get outboxEntries =>
      $$OutboxEntriesTableTableManager(_db, _db.outboxEntries);
}

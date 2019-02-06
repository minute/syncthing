// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'configuration.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Configuration _$ConfigurationFromJson(Map<String, dynamic> json) {
  return Configuration()
    ..version = json['version'] as int
    ..folders = (json['folders'] as List)
        ?.map((e) => e == null
            ? null
            : FolderConfiguration.fromJson(e as Map<String, dynamic>))
        ?.toList()
    ..devices = (json['devices'] as List)
        ?.map((e) => e == null
            ? null
            : DeviceConfiguration.fromJson(e as Map<String, dynamic>))
        ?.toList();
}

Map<String, dynamic> _$ConfigurationToJson(Configuration instance) =>
    <String, dynamic>{
      'version': instance.version,
      'folders': instance.folders,
      'devices': instance.devices
    };

FolderConfiguration _$FolderConfigurationFromJson(Map<String, dynamic> json) {
  return FolderConfiguration()
    ..id = json['id'] as String
    ..label = json['label'] as String
    ..path = json['path'] as String
    ..type = json['type'] as String;
}

Map<String, dynamic> _$FolderConfigurationToJson(
        FolderConfiguration instance) =>
    <String, dynamic>{
      'id': instance.id,
      'label': instance.label,
      'path': instance.path,
      'type': instance.type
    };

DeviceConfiguration _$DeviceConfigurationFromJson(Map<String, dynamic> json) {
  return DeviceConfiguration()
    ..deviceID = json['deviceID'] as String
    ..name = json['name'] as String
    ..addresses =
        (json['addresses'] as List)?.map((e) => e as String)?.toList();
}

Map<String, dynamic> _$DeviceConfigurationToJson(
        DeviceConfiguration instance) =>
    <String, dynamic>{
      'deviceID': instance.deviceID,
      'name': instance.name,
      'addresses': instance.addresses
    };

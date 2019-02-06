import 'package:json_annotation/json_annotation.dart';

part "configuration.g.dart";

@JsonSerializable()
class Configuration {
  int version;
  List<FolderConfiguration> folders;
  List<DeviceConfiguration> devices;
  // ...

  Configuration();

  factory Configuration.fromJson(Map<String, dynamic> json) =>
      _$ConfigurationFromJson(json);

  Map<String, dynamic> toJson() => _$ConfigurationToJson(this);
}

@JsonSerializable()
class FolderConfiguration {
  String id;
  String label;
  String path;
  String type;
  // ...

  String get labelOrID => label.isEmpty ? id : label;

  FolderConfiguration();

  factory FolderConfiguration.fromJson(Map<String, dynamic> json) =>
      _$FolderConfigurationFromJson(json);

  Map<String, dynamic> toJson() => _$FolderConfigurationToJson(this);
}

@JsonSerializable()
class DeviceConfiguration {
  String deviceID;
  String name;
  List<String> addresses;
  // ...

  String get nameOrID => name.isEmpty ? deviceID.split('-').first : name;

  DeviceConfiguration();

  factory DeviceConfiguration.fromJson(Map<String, dynamic> json) =>
      _$DeviceConfigurationFromJson(json);

  Map<String, dynamic> toJson() => _$DeviceConfigurationToJson(this);
}

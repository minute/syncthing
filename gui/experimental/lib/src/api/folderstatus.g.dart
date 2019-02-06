// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'folderstatus.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

FolderStatus _$FolderStatusFromJson(Map<String, dynamic> json) {
  return FolderStatus()
    ..globalBytes = json['globalBytes'] as int
    ..localBytes = json['localBytes'] as int
    ..needBytes = json['needBytes'] as int
    ..pullErrors = json['pullErrors'] as int;
}

Map<String, dynamic> _$FolderStatusToJson(FolderStatus instance) =>
    <String, dynamic>{
      'globalBytes': instance.globalBytes,
      'localBytes': instance.localBytes,
      'needBytes': instance.needBytes,
      'pullErrors': instance.pullErrors
    };

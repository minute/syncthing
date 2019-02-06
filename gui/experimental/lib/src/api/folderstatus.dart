import 'package:json_annotation/json_annotation.dart';

part "folderstatus.g.dart";

@JsonSerializable()
class FolderStatus {
  int globalBytes;
  int localBytes;
  int needBytes;
  int pullErrors;
  // ...

  FolderStatus();

  double get localCompletion =>
      globalBytes == 0 ? 100 : localBytes / globalBytes;

  bool get errored => pullErrors > 0;

  factory FolderStatus.fromJson(Map<String, dynamic> json) =>
      _$FolderStatusFromJson(json);

  Map<String, dynamic> toJson() => _$FolderStatusToJson(this);
}

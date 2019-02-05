import 'package:angular/angular.dart';

import 'gauge_component.dart';

@Component(
  selector: 'folder',
  templateUrl: 'folder_component.html',
  directives: [
    GaugeComponent,
    NgIf,
  ],
)
class FolderComponent {
  @Input()
  FolderInfo folderInfo;

  List<Segment> get localSegments => [
        Segment(folderInfo.error ? "#dc3545" : _colorFor(folderInfo.local),
            folderInfo.local),
        Segment("#f0f0f0", 100 - folderInfo.local)
      ];

  List<Segment> get remoteSegments => [
        Segment(_colorFor(folderInfo.remote), folderInfo.remote),
        Segment("#f0f0f0", 100 - folderInfo.remote)
      ];

  String _colorFor(double completion) {
    if (completion >= 100.0) {
      return "#28a745";
    }
    return "#17a2b8";
  }
}

class FolderInfo {
  String label;
  double local;
  double remote;
  bool error;

  FolderInfo(this.label, this.local, this.remote, this.error);

  String get percent => "${local.toStringAsFixed(1)} %";

  FolderStatus get status {
    if (error) {
      return FolderStatus.errored;
    }
    if (local < 100) {
      return FolderStatus.syncing;
    }
    return FolderStatus.upToDate;
  }
}

class FolderStatus {
  final int idx;

  const FolderStatus(this.idx);

  static const upToDate = FolderStatus(0);
  static const syncing = FolderStatus(1);
  static const errored = FolderStatus(2);

  String toString() => {
        upToDate.idx: "Up to date",
        syncing.idx: "Syncing",
        errored.idx: "Errored",
      }[idx];

  String get textClass => {
        upToDate.idx: "text-success",
        syncing.idx: "text-info",
        errored.idx: "text-danger",
      }[idx];
}

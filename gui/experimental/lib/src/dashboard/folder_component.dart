import 'package:angular/angular.dart';

import 'colors.dart';
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

  String get statusString {
    switch (folderInfo.status) {
      case FolderStatus.upToDate:
        return "Up to date";
      case FolderStatus.syncing:
        return "Syncing";
      case FolderStatus.errored:
        return "Errored";
    }
  }

  String get statusTextClass {
    switch (folderInfo.status) {
      case FolderStatus.upToDate:
        return "text-success";
      case FolderStatus.syncing:
        return "text-info";
      case FolderStatus.errored:
        return "text-danger";
    }
  }

  List<Segment> get localSegments => [
        Segment(folderInfo.error ? Colors.danger : _colorFor(folderInfo.local),
            folderInfo.local),
        Segment(Colors.grey, 100 - folderInfo.local)
      ];

  List<Segment> get remoteSegments => [
        Segment(_colorFor(folderInfo.remote), folderInfo.remote),
        Segment(Colors.grey, 100 - folderInfo.remote)
      ];

  String _colorFor(double completion) {
    if (completion >= 100.0) {
      return Colors.success;
    }
    return Colors.info;
  }
}

class FolderInfo {
  String label;
  double local;
  double remote;
  bool error;

  FolderInfo(this.label, this.local, this.remote, this.error);

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

enum FolderStatus { upToDate, syncing, errored }

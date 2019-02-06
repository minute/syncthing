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

  String get stateString {
    switch (folderInfo.status) {
      case FolderState.upToDate:
        return "Up to date";
      case FolderState.syncing:
        return "Syncing";
      case FolderState.errored:
        return "Errored";
    }
  }

  String get stateTextClass {
    switch (folderInfo.status) {
      case FolderState.upToDate:
        return "text-success";
      case FolderState.syncing:
        return "text-info";
      case FolderState.errored:
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

  FolderState get status {
    if (error) {
      return FolderState.errored;
    }
    if (local < 100) {
      return FolderState.syncing;
    }
    return FolderState.upToDate;
  }
}

enum FolderState { upToDate, syncing, errored }

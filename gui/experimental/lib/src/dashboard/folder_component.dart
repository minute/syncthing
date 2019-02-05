import 'package:angular/angular.dart';

import 'gauge_component.dart';

@Component(
  selector: 'folder',
  templateUrl: 'folder_component.html',
  directives: [GaugeComponent],
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
}

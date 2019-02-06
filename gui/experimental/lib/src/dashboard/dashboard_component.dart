import 'package:angular/angular.dart';

import '../api/api.dart';
import 'device_component.dart';
import 'folder_component.dart';

@Component(
  selector: 'dashboard',
  templateUrl: 'dashboard_component.html',
  directives: [
    DeviceComponent,
    FolderComponent,
    NgFor,
  ],
)
class DashboardComponent implements OnInit {
  final API _api;

  Configuration _configuration;
  Map<String, FolderStatus> _folderStatus = {};

  DashboardComponent(this._api);

  void ngOnInit() async {
    _configuration = await _api.getConfiguration();
    for (var folderCfg in _configuration.folders) {
      _folderStatus[folderCfg.id] = await _api.getFolderStatus(folderCfg.id);
    }
  }

  FolderInfo folderInfo(FolderConfiguration folder) {
    final s = _folderStatus[folder.id];
    if (s == null) {
      return FolderInfo(folder.labelOrID, 0, 0, false);
    }
    return FolderInfo(folder.labelOrID, s.localCompletion, 0, s.errored);
  }

  Iterable<FolderInfo> get folders => _configuration?.folders?.map(folderInfo);

  Iterable<DeviceInfo> get devices => _configuration?.devices
      ?.map((cfg) => DeviceInfo(cfg.nameOrID, DeviceConnection.None, 0, 0, 0));
}

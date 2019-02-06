import 'package:angular/angular.dart';

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
class DashboardComponent {
  List<FolderInfo> folders = [
    FolderInfo("Archive", 20, 100, false),
    FolderInfo("Documents", 55, 78, true),
    FolderInfo("Foto", 100, 90, false)
  ];
  List<DeviceInfo> devices = [
    DeviceInfo(DeviceConnection.Direct, 75, 1234, 5678),
    DeviceInfo(DeviceConnection.None, 25, 0, 0),
    DeviceInfo(DeviceConnection.Relay, 90, 2345, 6789),
    DeviceInfo(DeviceConnection.None, 100, 0, 0),
    DeviceInfo(DeviceConnection.Direct, 100, 0, 0),
  ];
}

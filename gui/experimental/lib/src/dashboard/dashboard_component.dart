import 'package:angular/angular.dart';

import 'folder_component.dart';

@Component(
  selector: 'dashboard',
  styleUrls: ['dashboard_component.css'],
  templateUrl: 'dashboard_component.html',
  directives: [
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
}

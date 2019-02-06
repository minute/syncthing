import 'package:angular/angular.dart';

import 'src/api/api.dart';
import 'src/dashboard/dashboard_component.dart';

@Component(
  selector: 'my-app',
  styleUrls: ['app_component.css'],
  templateUrl: 'app_component.html',
  directives: [DashboardComponent],
  providers: [API],
)
class AppComponent {
  // Nothing here yet. All logic is in TodoListComponent.
}

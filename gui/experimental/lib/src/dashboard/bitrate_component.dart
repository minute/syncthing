import 'package:angular/angular.dart';

@Component(
  selector: 'bitrate',
  templateUrl: 'bitrate_component.html',
  directives: [NgIf],
)
class BitrateComponent {
  static bool useBytes = false;

  @Input()
  double kbps = 0;

  @Input()
  String direction;

  String get rate {
    if (useBytes) {
      final r = kbps / 8;
      return "${r.toStringAsFixed(1)} KiB/s";
    }
    return "${kbps.toStringAsFixed(1)} kbps";
  }

  void toggleUnit() {
    useBytes = !useBytes;
  }
}

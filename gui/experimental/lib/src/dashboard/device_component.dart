import 'package:angular/angular.dart';

import 'bitrate_component.dart';
import 'colors.dart';
import 'gauge_component.dart';

@Component(
  selector: 'device',
  templateUrl: 'device_component.html',
  directives: [
    BitrateComponent,
    GaugeComponent,
    NgIf,
  ],
)
class DeviceComponent {
  static const activityCutoffKbps = 50;

  @Input()
  DeviceInfo deviceInfo;

  bool get connected => deviceInfo.connection != DeviceConnection.None;
  bool get viaRelay => deviceInfo.connection == DeviceConnection.Relay;
  bool get idle =>
      deviceInfo.downKbps < activityCutoffKbps &&
      deviceInfo.downKbps < activityCutoffKbps;

  String get action {
    if (idle) {
      return "Idle";
    }
    if (deviceInfo.upKbps > activityCutoffKbps &&
        deviceInfo.upKbps > 5 * deviceInfo.downKbps) {
      return "Uploading";
    }
    if (deviceInfo.downKbps > activityCutoffKbps &&
        deviceInfo.downKbps > 5 * deviceInfo.upKbps) {
      return "Downloading";
    }
    return "Up- & downloading";
  }

  String get statusString {
    if (deviceInfo.connection == DeviceConnection.None) {
      return "Disconnected";
    }
    if (deviceInfo.completion < 100) {
      return "Syncing";
    }
    return "Up to date";
  }

  String get statusTextClass {
    if (deviceInfo.connection == DeviceConnection.None) {
      return "text-secondary";
    }
    if (deviceInfo.completion < 100) {
      return "text-info";
    }
    return "text-success";
  }

  String get completionColor {
    if (deviceInfo.connection == DeviceConnection.None) {
      return Colors.secondary;
    }
    if (deviceInfo.completion >= 100.0) {
      return Colors.success;
    }
    //if (deviceInfo.connection == DeviceConnection.Direct) {
    return Colors.info;
    //}
    //return Colors.warning;
  }

  List<Segment> get segments => [
        Segment(completionColor, deviceInfo.completion),
        Segment(Colors.grey, 100 - deviceInfo.completion)
      ];
}

class DeviceInfo {
  final DeviceConnection connection;
  final double completion;
  final double downKbps; // kilobits per second
  final double upKbps;
  DeviceInfo(this.connection, this.completion, this.downKbps, this.upKbps);
}

enum DeviceConnection { None, Direct, Relay }

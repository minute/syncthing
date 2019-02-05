import 'dart:math';
import 'package:angular/angular.dart';

@Component(
  selector: 'gauge',
  templateUrl: 'gauge_component.html',
  directives: [
    NgIf,
    NgFor,
  ],
)
class GaugeComponent {
  static const radius = 100;
  static const circumference = 2 * pi * radius;
  static final height = 1.3 * radius;
  static final width = 2 * height;
  static final segmentWidth = radius / 2;

  String get viewBox => "0 0 ${width} ${height}";

  @Input()
  String title = "";

  @Input()
  bool invert = false;

  double get centerY => invert ? 0 : height;

  List<SizedSegment> _segments = [];
  List<SizedSegment> get sizedSegments => _segments;
  List<Segment> get segments => _segments;

  @Input()
  set segments(List<Segment> s) {
    _segments = s
        .where((seg) => seg.size > 0)
        .map((seg) => SizedSegment(seg.color, seg.size))
        .toList();

    final sum = _segments.fold(0, (sum, seg) => sum + seg.size);
    var curPos = circumference / 2;
    for (var seg in _segments) {
      var len = circumference / 2 * seg.size / sum;
      seg.length = len;
      seg.remain = circumference - len;
      if (invert) {
        seg.start = curPos + len;
        curPos += len;
      } else {
        seg.start = curPos;
        curPos -= len;
      }
    }
  }
}

class Segment {
  String color;
  num size;

  Segment(this.color, this.size);
}

class SizedSegment extends Segment {
  double start;
  double length;
  double remain;

  String get dashArray => "${length} ${remain}";

  SizedSegment(color, size) : super(color, size);
}

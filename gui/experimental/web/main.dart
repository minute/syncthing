import 'package:angular/angular.dart';
import 'package:experimental/app_component.template.dart' as ng;
import 'package:http/http.dart';
import 'package:http/browser_client.dart';

import 'main.template.dart' as self;

// This provides an injectable (http)Client by calling browserClientFactory
@GenerateInjector([
  const FactoryProvider(Client, browserClientFactory),
])
final InjectorFactory injector = self.injector$Injector;

void main() {
  runApp(ng.AppComponentNgFactory, createInjector: injector);
}

Client browserClientFactory() {
  final c = BrowserClient();
  c.withCredentials = true;
  return c;
}

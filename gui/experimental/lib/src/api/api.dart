import 'dart:core';
import 'dart:convert';
import 'dart:html';
import 'dart:js';

import 'package:angular/core.dart';
import 'package:http/http.dart';

import 'configuration.dart';
import 'folderstatus.dart';

export 'configuration.dart';
export 'folderstatus.dart';

@Injectable()
class API {
  final Client _client;
  String _urlBase;
  String _deviceID;
  String _csrfCookieName;
  String _csrfHeaderName;
  String _csrfToken;

  API(this._client) {
    // Figure out our device ID from a global Javascript object, and by that
    // also the names of our CSRF stuff.
    _deviceID = context['metadata']['deviceID'] as String;
    final shortDeviceID = _deviceID.substring(0, 5);
    _csrfCookieName = "CSRF-Token-${shortDeviceID}";
    _csrfHeaderName = "X-CSRF-Token-${shortDeviceID}";

    final loc = window.location;
    _urlBase = "${loc.protocol}//${loc.host}${loc.pathname}";
    while (_urlBase.endsWith("/")) {
      _urlBase = _urlBase.substring(0, _urlBase.length - 1);
    }
  }

  Future<Configuration> getConfiguration() async {
    final resp = await _get('/rest/system/config');
    return Configuration.fromJson(json.decode(resp.body));
  }

  Future<FolderStatus> getFolderStatus(String folderID) async {
    final resp =
        await _get('/rest/db/status?folder=${Uri.encodeComponent(folderID)}');
    return FolderStatus.fromJson(json.decode(resp.body));
  }

// Helpers

  Future<Response> _get(String path) async {
    return await _do("GET", path);
  }

  Future<Response> _post(String path, String body) async {
    return await _do("POST", path, body: body);
  }

  Future<Response> _put(String path, String body) async {
    return await _do("PUT", path, body: body);
  }

  Future<Response> _delete(String path) async {
    return await _do("DELETE", path);
  }

  Future<Response> _do(String method, String path, {String body}) async {
    final uri = Uri.parse(_urlBase + path);
    final req = Request(method, uri);

    if (body != null) {
      req.body = body;
      req.headers["Content-Type"] = "application/json; charset=utf-8";
    }

    _setCsrfHeader(req);

    final streamResp = await _client.send(req);
    if (streamResp.statusCode == 401) {
      throw NotLoggedInException();
    } else if (streamResp.statusCode > 299) {
      final resp = await Response.fromStream(streamResp);
      throw APIException(
          streamResp.statusCode, streamResp.reasonPhrase, resp.body);
    }
    return Response.fromStream(streamResp);
  }

  void _setCsrfHeader(Request req) {
    if (_csrfToken == null) {
      _csrfToken = _getCsrfToken();
    }
    if (_csrfToken != null) {
      req.headers[_csrfHeaderName] = _csrfToken;
    }
  }

  String _getCsrfToken() {
    final cookies = document.cookie.split(';').map((c) => c.trim());
    for (var cookie in cookies) {
      final fields = cookie.split('=');
      if (fields.length != 2) {
        continue;
      }
      if (fields[0] == _csrfCookieName) {
        return fields[1];
      }
    }
    return null;
  }
}

class NotLoggedInException implements Exception {
  String toString() {
    return "Authentication refused";
  }
}

class APIException implements Exception {
  final int statusCode;
  final String reasonPhrase;
  final String message;

  APIException(this.statusCode, this.reasonPhrase, this.message);

  String toString() {
    return message ?? "$statusCode $reasonPhrase";
  }
}

# dev/infrastructure.md

This is a list of the infrastructure that powers the Syncthing project. Unless otherwise noted, the default is that it\'s a VM hosted by `calmh`{.interpreted-text role="user"}.

## GitHub

All repos, issue trackers and binary releases are hosted at [GitHub](https://github.com/syncthing).

## Main & Documentation Websites

Static HTML, served by Caddy.

* [syncthing.net](https://syncthing.net/)
* [apt.syncthing.net](https://apt.syncthing.net) \(Debian packages\)
* [docs.syncthing.net](https://docs.syncthing.net/) \(Sphinx for site

  generation\)

* [upgrades.syncthing.net](https://upgrades.syncthing.net/meta.json)

  \(upgrade metadata\)

## Forum Website

Powered by Discourse.

* [forum.syncthing.net](https://forum.syncthing.net/)

## Global Discovery Servers

Runs the `stdiscosrv` instances that serve global discovery requests. The discovery setup is a load balanced cluster and the members can change without prior notice. As of the time of writing they are all hosted at DigitalOcean.

* discovery.syncthing.net \(multiple A and AAAA records, for queries\)
* discovery-v4.syncthing.net \(multiple A records, for IPv4 announces\)
* discovery-v6.syncthing.net \(multiple AAAA records, for IPv6

  announces\)

## Relay Pool Server

Runs the [strelaypoolsrv](https://github.com/syncthing/syncthing/tree/master/cmd/strelaypoolsrv) daemon to handle dynamic registration and announcement of relays.

* [relays.syncthing.net](http://relays.syncthing.net)

## Relay Servers

Hosted by friendly people on the internet.

## Usage Reporting Server

Runs the [ursrv](https://github.com/syncthing/syncthing/tree/master/cmd/ursrv) daemon with PostgreSQL and Nginx.

* [data.syncthing.net](https://data.syncthing.net/)

## Build Servers

Runs TeamCity and does the core builds.

* [build.syncthing.net](https://build.syncthing.net/)

There are various build agents; Linux, Windows, and Mac. These are currently provided by `calmh`{.interpreted-text role="users"} or Kastelo.

## Signing Server

Signs and uploads the release bundles to GitHub.

* secure.syncthing.net

## Monitoring

The infrastructure is monitored and its status is publicly accessible on the following urls:

* [status.syncthing.net](https://status.syncthing.net) \(Apex Ping\)
* [monitor.syncthing.net](https://monitor.syncthing.net) \(Grafana\)


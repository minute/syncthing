Community Contributions {#contributions}
=======================

This page lists integrations, addons and packagings of Syncthing created
by the community. Like all documentation pages, it\'s wiki editable so
please do edit and add your own.

GUI Wrappers
------------

### Cross-platform {#contrib-all}

-   [Syncthing-GTK](https://github.com/syncthing/syncthing-gtk)
-   [QSyncthingTray](https://github.com/sieren/QSyncthingTray)
-   [syncthingtray](https://github.com/Martchus/syncthingtray)
-   [syncthing-tray](https://github.com/alex2108/syncthing-tray)

### Android

-   [syncthing-android](https://github.com/syncthing/syncthing-android)

    A wrapper app for the Syncthing binary.

-   [a-sync-browser](https://github.com/davide-imbriaco/a-sync-browser)

    Down- or uploads data from accessible devices, does not continuously
    keep a share in sync.

### Windows {#contrib-windows}

-   [SyncTrayzor](https://github.com/canton7/SyncTrayzor)

    Windows host for Syncthing. Installer, auto-start, built-in browser,
    tray icon, folder watcher, and more.

-   [SyncthingTray](https://github.com/iss0/SyncthingTray)

    Wrapper including a small interface to configure start on boot and
    puts Syncthing into the systray instead of a console window.

### Mac OS

-   [SyncthingBar](https://github.com/nhojb/SyncthingBar)
-   [BitBar plugin](https://github.com/sebw/bitbar-plugins)

### Linux

-   [Syncthing
    Icon](https://extensions.gnome.org/extension/989/syncthing-icon/)

    A GNOME Shell extension displaying a Syncthing status icon in the
    top bar.

-   [syncthing-quick-status](https://github.com/serl/syncthing-quick-status)

    Small bash application with minimal dependencies, for a simple
    colorful representation of the current status.

Packages and Bundlings
----------------------

### Debian / Ubuntu

-   Official packages: <https://apt.syncthing.net/>
-   Debian packages:
    [syncthing](https://packages.debian.org/search?keywords=syncthing),
    [syncthing-discosrv](https://packages.debian.org/search?keywords=syncthing-discosrv)
    and
    [syncthing-relaysrv](https://packages.debian.org/search?keywords=syncthing-relaysrv)
-   [Ubuntu PPA containing
    Syncthing-GTK](https://launchpad.net/~nilarimogard/+archive/ubuntu/webupd8/)

### Snap

On any of the [Linux distributions that support
snaps](https://snapcraft.io/docs/core/install): :

    $ snap install syncthing

If you want to help testing the upcoming release, and get the newer
features earlier, you can install the snap from the candidate channel:

> \$ snap install syncthing \--candidate

### Fedora / CentOS

For Fedora Syncthing is now in the official repo :
<https://src.fedoraproject.org/rpms/syncthing>

Unofficial [RPM repo of
Syncthing](https://copr.fedorainfracloud.org/coprs/daftaupe/syncthing/)
([sources](https://gitlab.com/daftaupe/syncthing-rpm))

### ArchLinux

-   Official Community Repository:
    [syncthing](https://www.archlinux.org/packages/?name=syncthing) and
    [syncthing-gtk](https://www.archlinux.org/packages/?name=syncthing-gtk)
-   Arch User Repository:
    [syncthing-discosrv](https://aur.archlinux.org/packages/syncthing-discosrv)
    and
    [syncthingtray](https://aur.archlinux.org/packages/syncthingtray)

### Docker

-   Dockerfiles for
    [Syncthing](https://github.com/firecat53/dockerfiles/tree/master/syncthing)
    and [Syncthing Discovery
    Server](https://github.com/firecat53/dockerfiles/tree/master/syncthing_discovery).
    Latest binary releases used for both.
-   [docker-syncthing](https://github.com/joeybaker/docker-syncthing)

    A fully baked docker container that allows custom config and will
    keep your settings and data past docker image restarts.

-   [syncthing-docker-scratch](https://github.com/djtm/syncthing-docker-scratch)

    Builds docker containers from scratch base and/or runs the
    containers in docker or rkt.

-   [rpi-syncthing](https://github.com/funkyfuture/docker-rpi-syncthing)

    Configurable image for the Raspberry Pi.

### Gentoo

Official net-p2p package:
[syncthing](https://packages.gentoo.org/packages/net-p2p/syncthing)

### FreeBSD

FreshPorts: [syncthing](https://www.freshports.org/net/syncthing)

### OpenBSD

Official ports:
[syncthing](https://cvsweb.openbsd.org/cgi-bin/cvsweb/ports/net/syncthing)
and
[QSyncthingTray](https://cvsweb.openbsd.org/cgi-bin/cvsweb/ports/net/qsyncthingtray)

### OpenSUSE

Official packages:
[syncthing](https://software.opensuse.org/package/syncthing) and
[syncthingtray](https://software.opensuse.org/package/syncthingtray)

### Synology NAS (DSM)

Add `http://packages.synocommunity.com/` to the Package Center in DSM or
view the [browsable repository](https://synocommunity.com/packages).
Numerous CPU architectures are supported. SPK\'s may be older versions,
however you can execute a Syncthing version upgrade via the web GUI
after installation.

### QNAP NAS (QTS)

[Syncthing QPKG](https://forum.qnap.com/viewtopic.php?f=320&t=97035)
(Qnap Package) available for ALL models x86, x86\_64, Arm (all including
new models).

### RockStor

[Docker
container](http://rockstor.com/docs/docker-based-rock-ons/syncthing.html)
and [registry
entry](https://github.com/rockstor/rockon-registry/blob/master/syncthing.json)

### ClearOS / WikiSuite

Syncthing is part of [WikiSuite](http://wikisuite.org/), and thus
packaged for
[ClearOS](http://wikisuite.org/How-to-install-Syncthing-on-ClearOS).

### Cloudron

Syncthing is available as a 1-click install on
[Cloudron](https://cloudron.io). For those unaware, Cloudron makes it
easy to run apps on your server and keep them up-to-date and secure.

[![image](https://cloudron.io/img/button.svg)](https://cloudron.io/button.html?app=net.syncthing.cloudronapp2)

There is a [demo available](https://my-demo.cloudron.me) (username:
cloudron password: cloudron)

The Cloudron package is developed
[here](https://git.cloudron.io/cloudron/syncthing-app).

Integrations
------------

### REST API Bindings

-   Python: <https://github.com/blakev/python-syncthing>
    (<https://pypi.python.org/pypi/syncthing>)
-   PHP: <https://github.com/terzinnorbert/syncthing-rest>

### Ports

-   Swift:
    [pulse-swift](https://source.ind.ie/project/pulse-swift/tree/master)

    Currently still in development and is \"not yet usable by any
    standard\". Only the Block Exchange Protocol layer and the
    Connection layer are completed.

-   Java: [a-sync](https://github.com/davide-imbriaco/a-sync)

    This implements the BEP, discovery and relay protocols providing a
    command line utility to access a Syncthing network, a service to
    proxy the relay protocol over http and a client library for the BEP
    protocol.

### Configuration management

-   [puppet-syncthing](https://github.com/whefter/puppet-syncthing)
-   [ansible-syncthing](https://github.com/le9i0nx/ansible-syncthing)
-   Command line interface:
    [syncthingmanager](https://github.com/classicsc/syncthingmanager)

### Monitoring

-   [munin-syncthing](https://gitlab.com/daftaupe/munin-syncthing)

### Resolving conflicts

-   [syncthing-resolve-conflicts](https://github.com/dschrempf/syncthing-resolve-conflicts)

    A small bash script that handles synchronization conflicts in text
    files that may pop up when using Syncthing. It is inspired by the
    [pacdiff]{.title-ref} utility from Arch Linux. A diff utility can be
    used to merge the files and keep them up to date.

Older, Possibly Unmaintained
----------------------------

::: {.note}
::: {.admonition-title}
Note
:::

These projects have not been updated in quite a while. They may still be
usable, or they may be in disrepair. If you are the maintainer of one of
these and you have revived the project, please update this page
accordingly.
:::

-   <https://github.com/akissa/pysyncthing>
-   <https://github.com/retgoat/syncthing-ruby>
-   <https://github.com/sodacode/Windows-Syncthing-Installer>
-   <https://github.com/gutenye/syncthing-kindle>
-   <https://github.com/m0ppers/syncthing-bar> (OSX 10.10 only)
-   <https://github.com/graboluk/stiko>
-   <https://github.com/ALinuxNinja/salt-syncthing>
-   <https://www.asustor.com/apps/app_detail?id=552>
-   <https://susestudio.com/a/qkdvwb/syncthing>
-   <https://source.ind.ie/project/pulse-swift/tree/master>
-   <https://github.com/icaruseffect/syncthing-ubuntu-indicator>
-   <https://github.com/bloones/SyncThingWin>
-   <https://github.com/thunderbirdtr/syncthing_rpm>
-   <https://github.com/dapperstout/pulse-java>
-   <https://github.com/cebe/pulse-php-discover>

Starting Syncthing Automatically
================================

::: {.warning}
::: {.admonition-title}
Warning
:::

This page may be outdated and requires review.
:::

Jump to configuration for your system:

-   [Windows](#windows)
-   [macOS](#macos)
-   [Linux](#linux)

Windows
-------

There is currently no official installer available for Windows. However,
there are a number of easy solutions.

### Task Scheduler

1.  Start the [Task
    Scheduler](https://en.wikipedia.org/wiki/Windows_Task_Scheduler)
    (`taskschd.msc`)
2.  Create a New Task (\"Action\" menu -\> \"Create Task\...\")
3.  

    General Tab:

    :   1.  Name the task (for example \'Syncthing\')
        2.  Check \"Run whether user is logged on or not\"

4.  

    Triggers Tab:

    :   1.  Click \"New\...\"
        2.  Set \"Begin the task\" to \"At Startup\"
        3.  (optional) choose a delay
        4.  Make sure Enabled is checked
        5.  Click \"OK\"

5.  

    Actions Tab:

    :   1.  Click \"New\...\"
        2.  \[Action\] should be set as \"Start a program\"
        3.  Enter the path to syncthing.exe in \"Program/Script\"
        4.  (optional) Enter \"-no-console -no-browser\" for \"Add
            arguments (optional)\"
        5.  Click \"OK\"

6.  

    Settings Tab:

    :   1.  (recommended) Keep the checkbox on \"Allow task to be run on
            demand\"
        2.  Clear checkbox from \"Stop task if it runs longer than:\"
        3.  (recommended) Keep \"Do not start a new instance\" for \"If
            the task is already running, then the following rule
            applies\"

7.  Click OK
8.  Enter password for the user.

### Third-party Tools

There are a number of third-party utilities which aim to address this
issue. These typically provide an installer, let Syncthing start
automatically, and a more polished user experience (e.g. by behaving as
a \"proper\" Windows application, rather than forcing you to start your
browser to interact with Syncthing).

::: {.seealso}
`Windows GUI Wrappers <contrib-windows>`{.interpreted-text role="ref"},
`Cross-platform GUI Wrappers <contrib-all>`{.interpreted-text
role="ref"}.
:::

### Start on Login

Starting Syncthing on login, without a console window or browser opening
on start, is relatively easy.

1.  Find the correct link of the Windows binary from the [Syncthing
    website](https://github.com/syncthing/syncthing/releases) (choose
    **amd64** if you have a 64-bit version of Windows)
2.  Extract the files in the folder (`syncthing-windows-*`) in the zip
    to the folder `C:\syncthing`
3.  Go to the `C:\syncthing` folder, make a file named `syncthing.bat`
4.  Right-click the file and choose **Edit**. The file should open in
    Notepad or your default text editor.
5.  Paste the following command into the file and save the changes:
    `start "Syncthing" syncthing.exe -no-console -no-browser`
6.  Right-click on `syncthing.bat` and press \"Create Shortcut\"
7.  Right-click the shortcut file `syncthing.bat - Shortcut` and click
    **Copy**
8.  Click **Start**, click **All Programs**, then click **Startup**.
    Right-click on **Startup** then click **Open**. ![Setup
    Screenshot](st2.png)
9.  Paste the shortcut (right-click in the folder and choose **Paste**,
    or press `CTRL+V`)

Syncthing will now automatically start the next time you open a new
Windows session. No console or browser window will pop-up. Access the
interface by browsing to <http://localhost:8384/>

If you prefer slower indexing but a more responsive system during scans,
copy the following command instead of the command in step 5:

    start "Syncthing" /low syncthing.exe -no-console -no-browser

### Run as a service independent of user login

::: {.warning}
::: {.admonition-title}
Warning
:::

There are important security considerations with this approach. If you
do not secure Syncthing\'s GUI (and REST API), then **any** process
running with **any** permissions can read/write **any** file on your
filesystem, by opening a connection with Syncthing.

Therefore, you **must** ensure that you set a GUI password, or run
Syncthing as an unprivileged user.
:::

With the above configuration, Syncthing only starts when a user logs on
to the machine. This is not optimal on servers where a machine can run
long times after a reboot without anyone logged in. In this case it is
best to create a service that runs as soon as Windows starts. This can
be achieved using NSSM, the \"Non-Sucking Service Manager\".

Note that starting Syncthing on login is the preferred approach for
almost any end-user scenario. The only scenario where running Syncthing
as a service makes sense is for (mostly) headless servers, administered
by a sysadmin who knows enough to understand the security implications.

1.  Download and extract [nssm](http://nssm.cc/download) to a folder
    where it can stay. The NSSM executable performs administration as
    well as executing as the Windows service so it will need to be kept
    in a suitable location.
2.  From an administrator Command Prompt, CD to the NSSM folder and run
    `nssm.exe install <syncthing service name>`
3.  Application Tab
    -   Set *Path* to your `syncthing.exe` and enter
        `-no-restart -no-browser -home="<path to your Syncthing folder>"`
        as Arguments. Note: Logging is set later on. `-logfile` here
        will not be applied.
    -   ![Windows NSSM Configuration
        Screenshot](windows-nssm-config.png)
4.  Details Tab
    -   Optional: Set *Startup type* to *Automatic (Delayed Start)* to
        delay the start of Syncthing when the system first boots, to
        improve boot speed.
5.  Log On Tab
    -   Enter the user account to run Syncthing as. This user needs to
        have full access to the Syncthing executable and its parent
        folder, configuration files / database folder and synced
        folders. You can leave this as *Local System* but doing so poses
        security risks. Setting this to your Windows user account will
        reduce this; ideally create a dedicated user account with
        minimal permissions.
6.  Process Tab
    -   Optional: Change priority to *Low* if you want a more responsive
        system at the cost of somewhat longer sync time when the system
        is busy.
    -   Optional: To enable logging enable \"Console window\".
7.  Shutdown Tab
    -   To ensure Syncthing is shut down gracefully select all of the
        checkboxes and set all *Timeouts* to *10000ms*.
8.  Exit Actions Tab
    -   Set *Restart Action* to *Stop service (oneshot mode)*. Specific
        settings are used later for handling Syncthing exits, restarts
        and upgrades.
9.  I/O Tab
    -   Optional: To enable logging set *Output (stdout)* to the file
        desired for logging. The *Error* field will be automatically set
        to the same file.
10. File Rotation Tab
    -   Optional: Set the rotation settings to your preferences.
11. Click the *Install Service* Button
12. To ensure that Syncthing exits, restarts and upgrades are handled
    correctly by the Windows service manager, some final settings are
    needed. Execute these in the same Command Prompt:
    -   `nssm set syncthing AppExit Default Exit`
    -   `nssm set syncthing AppExit 0 Exit`
    -   `nssm set syncthing AppExit 3 Restart`
    -   `nssm set syncthing AppExit 4 Restart`
13. Start the service via `sc start syncthing` in the Command Prompt.
14. Connect to the Syncthing UI, enable HTTPS, and set a secure username
    and password.

macOS
-----

### Using [homebrew](https://brew.sh)

1.  `brew install syncthing`
2.  Follow the information presented by `brew` to autostart Syncthing
    using launchctl.

### Without homebrew

Download and extract Syncthing for Mac:
<https://github.com/syncthing/syncthing/releases/latest>.

1.  Copy the syncthing binary (the file you would open to launch
    Syncthing) into a directory called `bin` in your home directory i.e.
    into /Users/\<username\>/bin. If \"bin\" does not exist, create it.
2.  Open `syncthing.plist` located in /etc/macosx-launchd. Replace the
    four occurrences of /Users/USERNAME with your actual home directory
    location.
3.  Copy the `syncthing.plist` file to `~/Library/LaunchAgents`. If you
    have trouble finding this location select the \"Go\" menu in Finder
    and choose \"Go to folder\...\" and then type
    `~/Library/LaunchAgents`. Copying to \~/Library/LaunchAgents will
    require admin password in most cases.
4.  Log out and back in again. Or, if you do not want to log out, you
    can run this command in terminal:
    `launchctl load ~/Library/LaunchAgents/syncthing.plist`

**Note:** You probably want to turn off \"Start Browser\" in the web GUI
settings to avoid it opening a browser window on each login. Then, to
access the GUI type 127.0.0.1:8384 (by default) into Safari.

Linux
-----

### On any distribution (Arch, Debian, Linux Mint, Ubuntu, openSUSE)

1.  Launch the program \'Startup Applications\'.
2.  Click \'Add\'.
3.  Fill out the form:
    -   Name: Syncthing
    -   Command:
        `/path/to/syncthing/binary -no-browser -home="/home/your\_user/.config/syncthing"`

### Using Supervisord

Add the following to your supervisor config file:

    [program:syncthing]
    command = /path/to/syncthing/binary -no-browser -home="/home/some_user/.config/syncthing"
    directory = /home/some_user/
    autorestart = True
    user = some_user
    environment = STNORESTART="1", HOME="/home/some_user"

The file is located at `/etc/supervisor/supervisord.conf`
(Debian/Ubuntu) or `/etc/supervisord.conf` .

### Using systemd

systemd is a suite of system management daemons, libraries, and
utilities designed as a central management and configuration platform
for the Linux computer operating system. It also offers users the
ability to manage services under the user\'s control with a per-user
systemd instance, enabling users to start, stop, enable, and disable
their own units. Service files for systemd are provided by Syncthing and
can be found in
[etc/linux-systemd](https://github.com/syncthing/syncthing/tree/master/etc/linux-systemd).

You have two primary options: You can set up Syncthing as a system
service, or a user service.

Running Syncthing as a system service ensures that Syncthing is run at
startup even if the Syncthing user has no active session. Since the
system service keeps Syncthing running even without an active user
session, it is intended to be used on a *server*.

Running Syncthing as a user service ensures that Syncthing only starts
after the user has logged into the system (e.g., via the graphical login
screen, or ssh). Thus, the user service is intended to be used on a
*(multiuser) desktop computer*. It avoids unnecessarily running
Syncthing instances.

Several distros (including Arch Linux) ship the needed service files
with the Syncthing package. If your distro provides a systemd service
file for Syncthing, you can skip step 2 when you setting up either the
system service or the user service, as described below.

#### How to set up a system service

1.  Create the user who should run the service, or choose an existing
    one.
2.  Copy the `Syncthing/etc/linux-systemd/system/syncthing@.service`
    file into the [load path of the system
    instance](https://www.freedesktop.org/software/systemd/man/systemd.unit.html#Unit%20File%20Load%20Path).
3.  Enable and start the service. Replace \"myuser\" with the actual
    Syncthing user after the `@`:

        systemctl enable syncthing@myuser.service
        systemctl start syncthing@myuser.service

#### How to set up a user service

1.  Create the user who should run the service, or choose an existing
    one. *Probably this will be your own user account.*
2.  Copy the `Syncthing/etc/linux-systemd/user/syncthing.service` file
    into the [load path of the user
    instance](https://www.freedesktop.org/software/systemd/man/systemd.unit.html#Unit%20File%20Load%20Path).
    To do this without root privileges you can just use this folder
    under your home directory: `~/.config/systemd/user/`.
3.  Enable and start the service:

        systemctl --user enable syncthing.service
        systemctl --user start syncthing.service

#### Checking the service status

To check if Syncthing runs properly you can use the `status` subcommand.
To check the status of a system service:

    systemctl status syncthing@myuser.service

To check the status of a user service:

    systemctl --user status syncthing.service

#### Using the journal

Systemd logs everything into the journal, so you can easily access
Syncthing log messages. In both of the following examples, `-e` tells
the pager to jump to the very end, so that you see the most recent logs.

To see the logs for the system service:

    journalctl -e -u syncthing@myuser.service

To see the logs for the user service:

    journalctl -e --user-unit=syncthing.service

#### Permissions

If you enabled the `Ignore Permissions` option in the Syncthing
client\'s folder settings, then you will also need to add the line
`UMask=0002` (or any other [umask setting
\<http://www.tech-faq.com/umask.html\>]{.title-ref} you like) in the
`[Service]` section of the `syncthing@.service` file.

#### Debugging

If you are asked on the bugtracker to start Syncthing with specific
environment variables it will not work the normal way. Systemd isolates
each service and it cannot access global environment variables. The
solution is to add the variables to the service file instead.

To edit the system service, run:

    systemctl edit syncthing@myuser.service

To edit the user service, run:

    systemctl --user edit syncthing.service

This will create an additional configuration file automatically and you
can define (or overwrite) further service parameters like e.g.
`Environment=STTRACE=model`.

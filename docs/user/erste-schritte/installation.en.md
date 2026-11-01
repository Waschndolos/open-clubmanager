# Installation

Open ClubManager is a desktop program. There is no web version: you install it on every computer you want to use it on.

## System requirements

| | Minimum |
|---|---|
| **Windows** | Windows 10 or 11 (64-bit) |
| **macOS** | macOS 11 Big Sur or later (Intel or Apple silicon) |
| **Linux** | a current distribution with WebKitGTK 4.1 (e.g. Ubuntu 22.04+, Debian 12+, Fedora 38+) |
| **Disk space** | about 50 MB for the program, plus a few MB for the club data |
| **Internet** | only for downloading, updates and collaboration |

## Download

The installers are on the project's [releases page](https://github.com/Waschndolos/open-clubmanager/releases).

!!! info "No ready-made installers yet"
    Until the first version is released, there are no installers there yet. If you are technically inclined, you can build the program yourself; see the [project README](https://github.com/Waschndolos/open-clubmanager#development).

=== "Windows"

    1. Download `Open ClubManager_x.y.z_x64-setup.exe` (or the `.msi`).
    2. Double-click it and follow the installer.
    3. If Windows SmartScreen warns you ("Windows protected your PC"), click **More info → Run anyway**.
    4. Start Open ClubManager from the Start menu.

=== "macOS"

    1. Download `Open ClubManager_x.y.z_universal.dmg`.
    2. Open it and drag Open ClubManager into the **Applications** folder.
    3. On first start: right-click the app → **Open**, if macOS refuses to start it.

=== "Linux"

    - **Debian/Ubuntu:** `sudo apt install ./open-clubmanager_x.y.z_amd64.deb`
    - **Fedora/openSUSE:** `sudo dnf install ./open-clubmanager-x.y.z.x86_64.rpm`
    - **Others:** make the `.AppImage` file executable (`chmod +x`) and run it.

## What gets installed?

Open ClubManager consists of the program window and a small background service that manages the data. The service starts and stops automatically with the window and can **only be reached from your own computer**.

Besides the program, Open ClubManager creates a configuration file:

| Operating system | Configuration location |
|---|---|
| Windows | `%AppData%\open-clubmanager\config.json` |
| macOS | `~/Library/Application Support/open-clubmanager/config.json` |
| Linux | `~/.config/open-clubmanager/config.json` |

The club data itself lives in the database file whose location you choose during [setup](einrichtung.md).

!!! tip "Turn on disk encryption"
    The database file is not additionally encrypted on your computer. Therefore turn on your operating system's encryption (Windows: BitLocker or device encryption, macOS: FileVault, Linux: LUKS). Especially on laptops, this protects member data if the device gets lost.

## Uninstalling

Uninstall Open ClubManager like any other program. **Your database file is kept.** Delete it yourself if you no longer need it, and don't forget any backups.

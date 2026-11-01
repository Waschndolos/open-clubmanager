# Installation

Open ClubManager ist ein Desktop-Programm. Es gibt keine Web-Version – du installierst es auf jedem Computer, auf dem du damit arbeiten möchtest.

## Systemvoraussetzungen

| | Mindestens |
|---|---|
| **Windows** | Windows 10 oder 11 (64 Bit) |
| **macOS** | macOS 11 Big Sur oder neuer (Intel oder Apple Silicon) |
| **Linux** | aktuelle Distribution mit WebKitGTK 4.1 (z. B. Ubuntu 22.04+, Debian 12+, Fedora 38+) |
| **Speicher** | ca. 50 MB für das Programm, dazu wenige MB für die Vereinsdaten |
| **Internet** | nur für Download, Updates und die Zusammenarbeit |

## Herunterladen

Die Installationsdateien findest du auf der [Release-Seite des Projekts](https://github.com/Waschndolos/open-clubmanager/releases).

!!! info "Noch keine fertigen Installer"
    Solange es noch keine erste Version gibt, sind dort noch keine Installationsdateien. Wer technisch versiert ist, kann das Programm selbst bauen – siehe [README des Projekts](https://github.com/Waschndolos/open-clubmanager#development).

=== "Windows"

    1. Lade die Datei `Open ClubManager_x.y.z_x64-setup.exe` (oder `.msi`) herunter.
    2. Doppelklicke darauf und folge dem Installationsassistenten.
    3. Falls Windows SmartScreen warnt („Der Computer wurde durch Windows geschützt“), klicke auf **Weitere Informationen → Trotzdem ausführen**.
    4. Starte Open ClubManager über das Startmenü.

=== "macOS"

    1. Lade die Datei `Open ClubManager_x.y.z_universal.dmg` herunter.
    2. Öffne sie und ziehe Open ClubManager in den Ordner **Programme**.
    3. Beim ersten Start: Rechtsklick auf das Programm → **Öffnen**, falls macOS den Start verweigert.

=== "Linux"

    - **Debian/Ubuntu:** `sudo apt install ./open-clubmanager_x.y.z_amd64.deb`
    - **Fedora/openSUSE:** `sudo dnf install ./open-clubmanager-x.y.z.x86_64.rpm`
    - **Andere:** Die `.AppImage`-Datei ausführbar machen (`chmod +x`) und starten.

## Was wird installiert?

Open ClubManager besteht aus einem Programmfenster und einem kleinen Hintergrunddienst, der die Daten verwaltet. Der Dienst wird automatisch mit dem Fenster gestartet und beendet und ist **nur von deinem eigenen Computer** aus erreichbar.

Neben dem Programm legt Open ClubManager eine Konfigurationsdatei an:

| Betriebssystem | Ort der Konfiguration |
|---|---|
| Windows | `%AppData%\open-clubmanager\config.json` |
| macOS | `~/Library/Application Support/open-clubmanager/config.json` |
| Linux | `~/.config/open-clubmanager/config.json` |

Die eigentlichen Vereinsdaten liegen in der Datenbankdatei, deren Ort du bei der [Einrichtung](einrichtung.md) selbst festlegst.

!!! tip "Festplattenverschlüsselung einschalten"
    Die Datenbankdatei ist auf deinem Computer nicht zusätzlich verschlüsselt. Schalte deshalb die Verschlüsselung deines Betriebssystems ein (Windows: BitLocker bzw. Geräteverschlüsselung, macOS: FileVault, Linux: LUKS). Gerade bei Laptops schützt das die Mitgliederdaten, falls das Gerät verloren geht.

## Deinstallieren

Deinstalliere Open ClubManager wie jedes andere Programm. **Deine Datenbankdatei bleibt dabei erhalten.** Lösche sie selbst, wenn du sie nicht mehr brauchst – und denke an eventuelle Datensicherungen.

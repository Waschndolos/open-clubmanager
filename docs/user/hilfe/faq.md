# Häufige Fragen

## Allgemein

??? question "Kostet Open ClubManager etwas?"
    Nein. Open ClubManager ist freie Software unter der Lizenz AGPL-3.0. Es gibt keine kostenpflichtige Version, keine Mitgliederobergrenze und kein Abo.

??? question "Gibt es eine Web-Version oder eine Handy-App?"
    Nein, und das ist auch nicht geplant. Open ClubManager ist bewusst ein Desktop-Programm: Die Daten bleiben auf euren Geräten, und niemand muss einen Server betreiben. Der Nachteil: Ohne einen Computer mit Open ClubManager kommst du nicht an die Daten.

??? question "Können Mitglieder ihre Daten selbst ändern?"
    Nein. Open ClubManager ist für die Ehrenamtlichen im Verein gedacht, nicht für die Mitglieder selbst.

??? question "Wie viele Mitglieder verträgt Open ClubManager?"
    Ausgelegt ist es für Vereine mit einigen Dutzend bis einigen Tausend Mitgliedern. Die Datenbank (SQLite) kommt auch mit deutlich mehr zurecht.

??? question "Brauche ich Internet?"
    Nur für Download, Updates und die Zusammenarbeit. Im Modus „Nur auf diesem Gerät“ arbeitest du komplett offline.

## Daten

??? question "Wo liegen meine Daten?"
    In der Datenbankdatei, deren Ort du bei der Einrichtung gewählt hast. Du siehst den Pfad in den [Einstellungen](../funktionen/einstellungen.md).

??? question "Kann ich die Datenbankdatei in meinen Dropbox-/OneDrive-Ordner legen, damit andere darauf zugreifen?"
    Bitte nicht. Gleichzeitige Synchronisation kann die Datei beschädigen, und jeder mit Zugriff auf den Ordner könnte alle Daten lesen. Nutze stattdessen die [Zusammenarbeit](../zusammenarbeit/index.md) – sie legt nur verschlüsselte Änderungsdateien in den Ordner.

??? question "Was passiert, wenn das Projekt eingestellt wird?"
    Das installierte Programm funktioniert weiter. Deine Daten liegen in einer offenen SQLite-Datei und lassen sich jederzeit exportieren. Siehe [Daten exportieren & mitnehmen](../migration/daten-mitnehmen.md).

??? question "Wie sichere ich meine Daten?"
    Über **Einstellungen → Datensicherung → Jetzt sichern**. Siehe [Datensicherung](../funktionen/einstellungen.md#datensicherung).

## Zusammenarbeit

??? question "Ich sehe die Änderungen meiner Kollegin nicht."
    Prüfe, ob das Sync-Programm deines Cloud-Anbieters läuft und der Ordner aktuell ist. Klicke dann im Sync-Status auf **Jetzt synchronisieren**. Neue Daten anderer kommen erst an, wenn dein Sync-Programm sie heruntergeladen hat.

??? question "Kann ich mehrere Vereine verwalten?"
    Im Moment nicht gleichzeitig mit Zusammenarbeit – pro Gerät wird ein Verein unterstützt. Im lokalen Modus kannst du in den [Einstellungen](../funktionen/einstellungen.md) zwischen Datenbankdateien wechseln.

??? question "Ein Laptop wurde gestohlen. Was tun?"
    Entferne das Gerät sofort unter [Personen & Geräte](../zusammenarbeit/personen-geraete.md#gerat-entfernen) und prüfe, ob eine Meldung an die Datenschutzaufsicht nötig ist (72 Stunden, Art. 33 DSGVO).

## Probleme melden

Fehler gefunden oder eine Idee? Melde dich im [Issue-Tracker auf GitHub](https://github.com/Waschndolos/open-clubmanager/issues). Bitte schicke dabei **keine echten Mitgliederdaten** mit.

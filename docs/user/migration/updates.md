# Updates & Datenbankversionen

## Ein Update installieren

1. **Datensicherung erstellen** ([Einstellungen → Jetzt sichern](../funktionen/einstellungen.md#datensicherung)).
2. Open ClubManager **schließen**.
3. Die neue Version von der [Release-Seite](https://github.com/Waschndolos/open-clubmanager/releases) herunterladen und **über die alte Version installieren**. Deine Datenbankdatei und Einstellungen bleiben erhalten.
4. Open ClubManager starten.
5. In den [Versionshinweisen](versionshinweise.md) nachsehen, ob die neue Version Handarbeit erfordert (selten).

## Was beim ersten Start passiert

Neue Versionen bringen manchmal neue Funktionen mit, für die die Datenbank erweitert werden muss (z. B. neue Tabellen für Haushalte oder Termine). Diese **Datenbank-Migration** läuft beim Start **automatisch**:

- Du musst nichts tun.
- Bestehende Daten werden nicht gelöscht, sondern übernommen.
- Bei großen Datenbanken kann der erste Start ein paar Sekunden länger dauern.

```
Version 1.2 → 1.3 → 1.4
  Datenbank   Schritt 13   Schritt 14   (werden beim Start nacheinander ausgeführt)
```

Du kannst Versionen auch **überspringen**: Von 1.1 direkt auf 1.4 zu aktualisieren ist kein Problem, alle fehlenden Schritte laufen nacheinander.

## Zurück zur alten Version?

Nach einer Migration kann eine **ältere** Programmversion die Datenbank nicht mehr zuverlässig öffnen. Willst du zurück:

1. Ältere Version installieren.
2. Die **Datensicherung von vor dem Update** einspielen.

Änderungen, die du nach dem Update gemacht hast, gehen dabei verloren.

## Updates bei der Zusammenarbeit

Bei der Zusammenarbeit tauschen alle Geräte Änderungen in einem gemeinsamen Format aus. Deshalb:

- **Alle Geräte zeitnah aktualisieren.** Ein Gerät mit einer **älteren** Version nimmt Änderungen von neueren Versionen nicht an und fordert zum Update auf. Seine eigenen Änderungen gehen dabei nicht verloren; sie werden übertragen, sobald es aktualisiert ist.
- Neuere Versionen verstehen Änderungen älterer Versionen immer.
- Absprache im Vorstand: Wer aktualisiert zuerst? Am besten alle innerhalb weniger Tage.

## Bei Problemen

- Startet das Programm nach dem Update nicht oder zeigt eine Fehlermeldung zur Datenbank: **nichts löschen**, Datensicherung bereithalten und im [Issue-Tracker](https://github.com/Waschndolos/open-clubmanager/issues) melden (ohne echte Mitgliederdaten).

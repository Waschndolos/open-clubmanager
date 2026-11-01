# Daten exportieren & mitnehmen

Deine Daten gehören dem Verein. Du kannst sie jederzeit aus Open ClubManager herausholen – auch wenn ihr zu einer anderen Software wechselt.

| Was | Wie | Format |
|---|---|---|
| Mitgliederliste | [Mitglieder → Export](../funktionen/mitglieder.md#excel-export) | Excel (`.xlsx`) |
| Daten eines Mitglieds | [Datenauskunft](../funktionen/mitglieder.md#datenauskunft-dsgvo) | PDF und JSON |
| Termine | [Kalender exportieren](../funktionen/termine.md#kalender-exportieren) | iCalendar (`.ics`) |
| Anwesenheit | [Auswertung exportieren](../funktionen/anwesenheit.md#auswertung) | CSV, PDF |
| Jubiläen | [Als CSV exportieren](../funktionen/ehrungen.md#anstehende-jubilaen) | CSV |
| **Alles** | [Datensicherung](../funktionen/einstellungen.md#datensicherung) | ZIP mit SQLite-Datenbank |

## Die Datenbank direkt öffnen

Die Datenbankdatei ist eine normale **SQLite**-Datenbank. Mit kostenlosen Werkzeugen wie [DB Browser for SQLite](https://sqlitebrowser.org/) kannst du sie öffnen und jede Tabelle als CSV exportieren – auch ohne Open ClubManager. Das ist der Weg für einen vollständigen Umzug in eine andere Software.

Die Tabellenstruktur ist im [Datenmodell](https://github.com/Waschndolos/open-clubmanager/blob/main/docs/data-model.md) beschrieben.

!!! warning
    Öffne die Datenbank nur, während Open ClubManager **geschlossen** ist, und arbeite am besten mit einer Kopie. Änderungen direkt in der Datenbank sind nicht vorgesehen und können bei der Zusammenarbeit zu Inkonsistenzen führen.

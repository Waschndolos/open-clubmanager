# Exporting your data

Your data belongs to the club. You can get it out of Open ClubManager at any time, also when switching to other software.

| What | How | Format |
|---|---|---|
| Member list | [Members → export](../funktionen/mitglieder.md#excel-export) | Excel (`.xlsx`) |
| One member's data | [Data export](../funktionen/mitglieder.md#data-export-gdpr) | PDF and JSON |
| Events | [Export calendar](../funktionen/termine.md#exporting-the-calendar) | iCalendar (`.ics`) |
| Attendance | [Export statistics](../funktionen/anwesenheit.md#statistics) | CSV, PDF |
| Anniversaries | [Export as CSV](../funktionen/ehrungen.md#upcoming-anniversaries) | CSV |
| **Everything** | [Backup](../funktionen/einstellungen.md#backup) | ZIP containing the SQLite database |

## Opening the database directly

The database file is a regular **SQLite** database. Free tools like [DB Browser for SQLite](https://sqlitebrowser.org/) can open it and export every table as CSV, even without Open ClubManager. That's the way to do a complete move to other software.

The table structure is described in the [data model](https://github.com/Waschndolos/open-clubmanager/blob/main/docs/data-model.md).

!!! warning
    Only open the database while Open ClubManager is **closed**, and preferably work on a copy. Editing the database directly is not supported and can cause inconsistencies with collaboration.

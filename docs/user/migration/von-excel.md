# Von Excel umsteigen

Die meisten Vereine starten mit einer Mitgliederliste in Excel. Mit dem Import übernimmst du sie in wenigen Minuten.

## 1. Excel-Liste vorbereiten

Open ClubManager liest `.xlsx`-Dateien. Die **erste Zeile** muss die Spaltenüberschriften enthalten, darunter ein Mitglied pro Zeile.

| Feld in Open ClubManager | Typische Spalte in deiner Liste |
|---|---|
| Mitgliedsnummer (`number`) | Mitgl.-Nr. |
| Vorname (`firstName`), Nachname (`lastName`) | Vorname, Name |
| E-Mail (`email`) – **Pflichtfeld** | E-Mail |
| Telefon (`phone`), Mobil (`phoneMobile`) | Tel., Handy |
| Geburtstag (`birthday`) | Geb.-Datum |
| Straße (`street`), PLZ (`postalCode`), Stadt (`city`), Bundesland (`state`) | Anschrift |
| Eintrittsdatum (`entryDate`), Austrittsdatum (`exitDate`) | Eintritt, Austritt |
| Kontoinhaber (`accountHolder`), IBAN (`iban`), BIC (`bic`), Bank (`bankName`) | Bankverbindung |
| SEPA-Mandat Datum (`sepaMandateDate`) | Mandat vom |
| Kommentar (`comment`) | Bemerkung |

Tipps:

- **Eine Spalte pro Feld.** Steht „Straße, PLZ Ort“ in einer Zelle, teile sie vorher in Excel auf (Daten → Text in Spalten).
- **Datumswerte** als echtes Excel-Datum formatieren, nicht als Text.
- **E-Mail ist Pflicht.** Mitglieder ohne E-Mail-Adresse bekommen einen eindeutigen Platzhalter (z. B. `mitglied-1042@verein.invalid`), den du später ersetzt.
- **Mitgliedsnummern und E-Mail-Adressen müssen eindeutig sein.** Doppelte Werte werden in der Vorschau markiert.
- Heißen deine Spalten genauso wie die englischen Feldnamen in Klammern, ordnet Open ClubManager sie automatisch zu.

## 2. Import ausführen

1. **Mitglieder → Import-Symbol** (:material-upload:) → **Excel-Datei auswählen**.
2. **Spalten zuordnen:** Für jede Spalte deiner Datei wählst du das passende Feld oder „Ignorieren“.
3. **Bestehende Mitglieder aktualisieren über:** E-Mail oder Mitgliedsnummer. Beim ersten Import ist das egal; bei einem späteren erneuten Import werden so vorhandene Mitglieder aktualisiert statt doppelt angelegt.
4. **Vorschau** prüfen und importieren.

## 3. Nacharbeiten

- **Abteilungen, Gruppen und Rollen** in der [Vereinsstruktur](../funktionen/vereinsstruktur.md) anlegen und über **Mitglieder zuweisen** gesammelt zuordnen.
- **Haushalte** für Familienmitgliedschaften anlegen ([Familien & Haushalte](../funktionen/haushalte.md)).
- **Beiträge** des laufenden Jahres unter [Finanzen → Mitgliedsbeiträge](../funktionen/finanzen.md#mitgliedsbeitrage) erfassen.
- **Stichproben** machen und dann die alte Excel-Liste sicher archivieren oder löschen.

## Weitere Listen

| Excel-Liste | Wohin |
|---|---|
| Kassenbuch | Ab einem Stichtag neu beginnen; Anfangsbestand als eine Einnahme-Buchung erfassen; Kontobewegungen per [CAMT.053-Import](../funktionen/finanzen.md#kontoauszug-importieren-camt053) |
| Inventarliste | in [Inventar](../funktionen/inventar.md) erfassen (noch kein Import) |
| Terminplan | in [Termine](../funktionen/termine.md) – Trainings als Serie anlegen |
| Lizenzen / Erste-Hilfe-Liste | in [Qualifikationen](../funktionen/qualifikationen.md) |

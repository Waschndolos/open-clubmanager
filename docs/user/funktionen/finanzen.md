---
routes:
  - /finance
---

# Finanzen

Kassenbuch und Mitgliedsbeiträge. Sichtbar nur für **Administratoren** und **Kassenwarte**.

![Kassenbuch](../assets/screenshots/finance.png){ .screenshot }

Oben siehst du den **Kontostand**, die **Gesamteinnahmen**, die **Gesamtausgaben** und die **offenen Beiträge**.

## Kassenbuch

Jede Einnahme und Ausgabe ist eine Buchung mit Datum, Beschreibung, Betrag, Typ (Einnahme/Ausgabe), Kategorie und Notizen.

- **Buchung hinzufügen** legt eine neue Buchung an.
- Über Stift und Papierkorb bearbeitest oder löschst du Buchungen.

Kosten aus der [Wartung im Inventar](inventar.md#prufung-wartung) können direkt als Ausgabe ins Kassenbuch gebucht werden.

### Kontoauszug importieren (CAMT.053)

Statt jede Buchung abzutippen, importierst du den Kontoauszug deiner Bank im Format **CAMT.053** (XML). Fast alle deutschen Banken bieten das im Online-Banking an, oft unter „Kontoauszug exportieren“ oder „Umsätze exportieren → CAMT“.

1. Klicke auf **CAMT.053 importieren** und wähle die Datei.
2. Open ClubManager legt für jede Kontobewegung eine Buchung an: Gutschriften als Einnahme, Lastschriften als Ausgabe.
3. Danach siehst du eine Zusammenfassung, wie viele Buchungen importiert wurden.

!!! warning "Derzeit noch zu beachten"
    - Importiere jeden Kontoauszug **nur einmal**. Bereits importierte Bewegungen werden noch nicht erkannt und würden doppelt gebucht.
    - Zahlungseingänge werden noch **nicht automatisch Mitgliedern zugeordnet**. Markiere bezahlte Beiträge unter **Mitgliedsbeiträge** selbst als bezahlt.

## Mitgliedsbeiträge

![Mitgliedsbeiträge](../assets/screenshots/finance-fees.png){ .screenshot }

Hier verwaltest du die Beiträge je Mitglied und Jahr: Betrag, Fälligkeitsdatum, Bezahlt-am-Datum und Beschreibung. Offene Beiträge sind als **Ausstehend** markiert.

## Kassenbuch komplett löschen

Für einen Neuanfang (z. B. nach dem Ausprobieren mit Beispieldaten) kannst du alle Buchungen und Beiträge auf einmal löschen. Zur Sicherheit musst du dazu ein Bestätigungswort eintippen. **Das lässt sich nicht rückgängig machen** – erstelle vorher eine [Datensicherung](einstellungen.md#datensicherung).

!!! warning "Kein Ersatz für die Buchhaltung"
    Das Kassenbuch ist eine Hilfe für den Vereinsalltag. Es ersetzt keine steuerlich anerkannte Buchführung, keinen DATEV-Export und keine Beratung durch Steuerberater oder Kassenprüfer.

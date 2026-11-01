---
routes:
  - /finance
---

# Finanzen

Kassenbuch, Mitgliedsbeiträge und Beitragseinzug per SEPA-Lastschrift. Sichtbar nur für **Administratoren** und **Kassenwarte**.

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

## Lastschrift (SEPA)

![Lastschriftläufe](../assets/screenshots/finance-direct-debit.png){ .screenshot }

Im Reiter **Lastschrift** ziehst du offene Beiträge per SEPA-Basislastschrift ein. Open ClubManager erstellt dafür eine Datei im Bankformat **pain.008**, die du im Online-Banking hochlädst. Der Reiter erscheint nur, wenn deine Rolle Finanzen bearbeiten darf und – bei der [Zusammenarbeit](../zusammenarbeit/index.md) – dein Gerät Bank- und Finanzdaten bekommt.

### Einmalig einrichten

1. In den [Einstellungen](einstellungen.md#sepa-lastschrift) unter **SEPA-Lastschrift** den Vereinsnamen, die **Gläubiger-ID** (von der Deutschen Bundesbank, z. B. `DE98ZZZ09999999999`), die **IBAN des Vereinskontos** und optional eine **Vorlage für den Verwendungszweck** hinterlegen.
2. Bei jedem Mitglied, das per Lastschrift zahlt, im [Mitglieder-Dialog](mitglieder.md#mitglied-anlegen-oder-bearbeiten) unter **Bankinformationen** IBAN, **SEPA Mandat Datum** und eine eindeutige **Mandatsreferenz** (z. B. `NW-1001`) eintragen.

### Lastschriftlauf erstellen

Klicke auf **Lastschriftlauf erstellen**. Ein Assistent führt in drei Schritten durch den Lauf:

1. **Beiträge auswählen:** Alle offenen Beiträge werden angezeigt. Beiträge, die sich nicht einziehen lassen (keine IBAN, kein Mandat, keine Mandatsreferenz, ungültige IBAN …), stehen unter **Nicht einziehbar** mit dem Grund – ergänze die Daten beim Mitglied.
2. **Fälligkeit & Verwendungszweck:** das **Fälligkeitsdatum (Einzug)**. Die Datei muss spätestens einen Bankarbeitstag vorher bei der Bank sein. Darunter siehst du eine Vorschau des Verwendungszwecks.
3. **Prüfen & exportieren:** Anzahl, Summe, Erst- (FRST) und Folgelastschriften (RCUR) prüfen und **Erstellen & herunterladen**.

Danach sind die Beiträge als **Im Einzug** markiert.

### Nach dem Export

| Schritt | Wo |
|---|---|
| Die `.xml`-Datei im Online-Banking hochladen und freigeben | bei deiner Bank |
| Wenn die Bank eingezogen hat: **Als bezahlt bestätigen** und das Datum angeben – alle Beiträge des Laufs werden als bezahlt markiert | Lastschriftlauf in der Liste |
| Rücklastschriften beim jeweiligen Beitrag nachtragen | [Mitgliedsbeiträge](#mitgliedsbeitrage) |
| Lauf versehentlich erstellt? **Lauf stornieren** – die Beiträge sind wieder offen. Die Datei dann nicht mehr hochladen bzw. den Auftrag bei der Bank widerrufen | Lastschriftlauf in der Liste |

Mit **XML-Datei herunterladen** bekommst du die Datei eines Laufs erneut.

!!! info "Erst- und Folgelastschrift"
    Wird ein Mandat zum ersten Mal genutzt, markiert Open ClubManager die Lastschrift als Erstlastschrift (FRST), danach als Folgelastschrift (RCUR). Das passiert automatisch.

## Kassenbuch komplett löschen

Für einen Neuanfang (z. B. nach dem Ausprobieren mit Beispieldaten) kannst du alle Buchungen und Beiträge auf einmal löschen. Zur Sicherheit musst du dazu ein Bestätigungswort eintippen. **Das lässt sich nicht rückgängig machen** – erstelle vorher eine [Datensicherung](einstellungen.md#datensicherung).

!!! warning "Kein Ersatz für die Buchhaltung"
    Das Kassenbuch ist eine Hilfe für den Vereinsalltag. Es ersetzt keine steuerlich anerkannte Buchführung, keinen DATEV-Export und keine Beratung durch Steuerberater oder Kassenprüfer.

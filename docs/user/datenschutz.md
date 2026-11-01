# Datenschutz

Vereine verarbeiten personenbezogene Daten – oft auch Bankdaten und Daten von Kindern. Open ClubManager hilft, die DSGVO einzuhalten. **Verantwortlich bleibt aber der Verein.**

!!! note "Keine Rechtsberatung"
    Diese Seite erklärt, wie Open ClubManager mit Daten umgeht und welche Funktionen dir helfen. Sie ersetzt keine Rechtsberatung. Viele Landessportbünde und Dachverbände bieten Muster für Datenschutzerklärungen und Verarbeitungsverzeichnisse an.

## Wo liegen die Daten?

| Modus | Wo | Wer kann zugreifen |
|---|---|---|
| **Nur auf diesem Gerät** | in der Datenbankdatei auf deinem Computer | wer Zugriff auf deinen Computer und dein Benutzerkonto hat |
| **Zusammenarbeit** | auf jedem beteiligten Gerät, dazu **verschlüsselt** im gemeinsamen Ordner | die eingeladenen Geräte – jeweils nur die Bereiche ihrer Rolle |

Open ClubManager schickt **keine** Daten an die Entwickler, an Analyse- oder Werbedienste. Es gibt keine Telemetrie.

### Cloud-Ordner bei der Zusammenarbeit

Im gemeinsamen Ordner liegen Mitglieder- und Bankdaten nur verschlüsselt. Der Cloud-Anbieter sieht aber **technische Metadaten**: Dateinamen, Dateigrößen, Zeitpunkte, Gerätenamen, die Rechteliste und Beitrittsanfragen. Prüfe, ob ihr mit dem Anbieter einen Auftragsverarbeitungsvertrag braucht – bei vereinseigener Nextcloud entfällt das meist.

## Schutz auf dem Gerät

Die Datenbank auf dem Gerät ist **nicht zusätzlich verschlüsselt**. Schütze sie deshalb:

- **Festplattenverschlüsselung** einschalten (BitLocker, FileVault, LUKS)
- Computer mit **Passwort** sperren
- **Datensicherungen** genauso schützen wie die Datenbank
- **Exporte** (Excel, CSV, PDF) nach Gebrauch löschen

## Funktionen für die DSGVO

| Pflicht | Funktion in Open ClubManager |
|---|---|
| **Auskunftsrecht** (Art. 15) | [Datenauskunft als PDF](funktionen/mitglieder.md#datenauskunft-dsgvo) je Mitglied |
| **Datenübertragbarkeit** (Art. 20) | Datenauskunft als JSON |
| **Berichtigung** (Art. 16) | Mitglied bearbeiten |
| **Löschung** (Art. 17) | Mitglied löschen |
| **Zugriffsbeschränkung** | [Benutzerrollen](erste-schritte/rollen.md) und [Geräterollen](zusammenarbeit/personen-geraete.md#rollen) |
| **Nachvollziehbarkeit** | [Verlauf](funktionen/verlauf.md) aller Änderungen |
| **Datenpanne** (Art. 33) | [Gerät entfernen](zusammenarbeit/personen-geraete.md#gerat-entfernen) mit Hinweis auf die 72-Stunden-Frist |
| **Führungszeugnisse** (§ 72a SGB VIII) | [Nur Einsichtsdatum speichern](funktionen/qualifikationen.md#fuhrungszeugnisse-72a-sgb-viii) |

## Bei der Zusammenarbeit beachten

Daten, die ein Gerät einmal bekommen hat, können nicht mehr „zurückgeholt“ werden – weder durch Rollenänderung noch durch Entfernen des Geräts. Das gilt für jede Software. Lade deshalb nur Personen ein, die die Daten für ihr Amt wirklich brauchen, und wähle die Rolle mit den geringsten nötigen Rechten.

# Benutzer & Rollen

Open ClubManager kennt zwei Arten von Rechten, die zusammenwirken:

1. **Benutzerrolle** – mit welchem Konto du dich anmeldest und was dieses Konto darf. Gilt immer.
2. **Geräterolle** – nur bei eingeschalteter [Zusammenarbeit](../zusammenarbeit/index.md): welche Datenbereiche dieses Gerät überhaupt bekommt.

Es gilt immer die strengere der beiden Regeln.

## Benutzerrollen

| Rolle | Darf |
|---|---|
| **Administrator** | alles, inklusive Benutzerverwaltung |
| **Kassenwart** | Vereinsdaten bearbeiten, Finanzen sehen und bearbeiten |
| **Schriftführer** | Vereinsdaten bearbeiten, Qualifikationen sehen und pflegen; keine Finanzen |
| **Trainer** | alles ansehen wie „Nur lesen“; zusätzlich Anwesenheit für die eigenen Gruppen und Abteilungen erfassen |
| **Nur lesen** | Daten ansehen, nichts ändern; keine Finanzen |

„Vereinsdaten“ sind Mitglieder, Vereinsstruktur, Inventar, Termine, Haushalte und Aufgaben.

„Keine Finanzen“ heißt: kein Kassenbuch, keine Beiträge und Beitragsmodelle, keine Lastschriften, kein Zahlungsabgleich, keine Zuwendungsbestätigungen – und im [Verlauf](../funktionen/verlauf.md) keine Einträge dazu.

!!! info "Wie findet Open ClubManager die Gruppen eines Trainers?"
    Ein Konto mit der Rolle **Trainer** wird über die **E-Mail-Adresse** mit einem Mitglied verknüpft. Die Gruppen und Abteilungen dieses Mitglieds bestimmen, für welche Termine der Trainer Anwesenheit erfassen darf.

Benutzer legst du in der [Benutzerverwaltung](../funktionen/benutzer.md) an.

## Geräterollen bei der Zusammenarbeit

| Bereich | Vorstand | Kassierer | Schriftführer | Übungsleiter | Gerätewart |
|---|---|---|---|---|---|
| Mitglieder | bearbeiten | sehen | bearbeiten | sehen | sehen |
| Kontaktdaten | bearbeiten | sehen | bearbeiten | – | – |
| Bankdaten | bearbeiten | bearbeiten | – | – | – |
| Finanzen | bearbeiten | bearbeiten | – | – | – |
| Inventar | bearbeiten | sehen | sehen | sehen | bearbeiten |
| Verwaltung | bearbeiten | – | – | – | – |

„–“ bedeutet: Das Gerät **bekommt diese Daten gar nicht**. Sie werden verschlüsselt übertragen, und das Gerät hat den Schlüssel nicht. Mehr dazu unter [Personen & Geräte](../zusammenarbeit/personen-geraete.md).

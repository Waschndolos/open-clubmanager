---
routes:
  - /collaboration/recover
---

# Notfallschlüssel

Der Notfallschlüssel wird beim [Einrichten des Vereins](einrichten.md) einmalig erzeugt. Mit ihm kommt ihr wieder an eure Vereinsdaten und die Verwaltung, wenn **kein Gerät mit der Rolle Vorstand** mehr verfügbar ist – zum Beispiel, weil der einzige Vorstandslaptop gestohlen wurde.

## Aufbewahren

- Bewahre ihn auf wie einen **Tresorschlüssel**: ausgedruckt im Vereinsordner, im Bankschließfach, beim Vorsitzenden zu Hause.
- **Nicht** im gemeinsamen Cloud-Ordner speichern – wer Ordner und Notfallschlüssel hat, hat vollen Zugriff.
- Niemand sonst kann ihn wiederherstellen – auch nicht die Entwickler von Open ClubManager.

## Verwenden

Nur verwenden, wenn wirklich kein Vorstandsgerät mehr da ist.

1. Installiere Open ClubManager auf einem Gerät und starte die Einrichtung mit **Notfallschlüssel verwenden** (oder unter **Zusammenarbeit → Notfall**).
2. Füge den ganzen Text des Notfallschlüssels ein oder tippe ihn ab. Wichtig sind die Zeilen `club-id`, `root-sign-key`, `root-age-key` und `check`.
3. Klicke auf **Verein wiederherstellen**.

Dieses Gerät ist danach ein **Vorstandsgerät**. Entferne als Nächstes unter [Personen & Geräte](personen-geraete.md#gerat-entfernen) alle Geräte, die verloren gegangen sind.

Der Notfallschlüssel bleibt gültig – bewahre ihn wieder sicher auf.

## Häufige Fehler

| Meldung | Bedeutung |
|---|---|
| „Der Notfallschlüssel ist unvollständig oder enthält einen Tippfehler“ | Text prüfen, besonders die Zeile `check` |
| „Dieser Notfallschlüssel gehört zu einem anderen Verein“ | falscher Schlüssel oder falscher Ordner |

# Von anderer Vereinssoftware umsteigen

Es gibt noch keinen direkten Import aus anderen Vereinsprogrammen. Der Weg führt über eine **Excel-Datei**: Fast jede Vereinssoftware kann die Mitgliederliste als Excel oder CSV exportieren.

## Allgemeines Vorgehen

1. **Export in der alten Software:** Mitgliederliste mit allen Feldern als Excel (`.xlsx`) oder CSV exportieren. Bankdaten und SEPA-Mandate mit exportieren, wenn ihr Lastschriften nutzt.
2. **CSV in Excel/LibreOffice öffnen und als `.xlsx` speichern.** Achte beim Öffnen auf die Zeichenkodierung (UTF-8), damit Umlaute stimmen.
3. **Aufbereiten** wie unter [Von Excel umsteigen](von-excel.md#1-excel-liste-vorbereiten) beschrieben.
4. **Importieren** und **prüfen**.
5. **Parallelbetrieb:** Lass die alte Software eine Weile (z. B. bis zum nächsten Beitragseinzug) nur lesend weiterlaufen, bevor du sie kündigst.

## Was sich nicht übernehmen lässt

- **Buchungshistorie** vergangener Jahre – schließe das alte Geschäftsjahr in der alten Software ab und beginne in Open ClubManager mit dem Anfangsbestand.
- **Dokumente und Anhänge** (Scans, Aufnahmeanträge) – Open ClubManager speichert keine Dateien. Bewahre sie im Vereinsarchiv auf.
- **Online-Zugänge für Mitglieder** – gibt es in Open ClubManager nicht.

## Anleitungen für bestimmte Programme

!!! info "Vorbereitet für Beiträge aus der Community"
    Hier sammeln wir Schritt-für-Schritt-Anleitungen für einzelne Programme. Du bist von einer anderen Software umgestiegen? Ergänze eine Anleitung – die Vorlage findest du unten und in [Dokumentation pflegen](../mitwirken/dokumentation.md).

| Software | Status |
|---|---|
| *noch keine* | – |

??? abstract "Vorlage für eine Anleitung"
    ```markdown
    ### Von <Name der Software>

    **Getestet mit:** <Version der alten Software>, Open ClubManager <Version>

    1. Export: <Menüpfad in der alten Software>, Format <xlsx/csv>
    2. Besonderheiten der Datei: <Trennzeichen, Kodierung, Datumsformat>
    3. Spaltenzuordnung:

    | Spalte im Export | Feld in Open ClubManager |
    |---|---|
    | … | … |

    4. Was fehlt / Nacharbeit: <…>
    ```

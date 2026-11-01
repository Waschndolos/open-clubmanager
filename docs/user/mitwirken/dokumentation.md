# Dokumentation pflegen

Dieses Handbuch wächst mit jeder neuen Funktion. Diese Seite richtet sich an alle, die an Open ClubManager mitarbeiten.

## Grundsatz

**Jede Funktion, die Nutzer sehen, ist dokumentiert – auf Deutsch und Englisch, im selben Pull Request.** CI prüft das automatisch.

## Aufbau

```
mkdocs.yml                         Konfiguration, Navigation und Übersetzung der Navigation
docs/requirements.txt              Versionen von MkDocs, Material-Theme und i18n-Plugin
docs/user/                         Quelltext dieses Handbuchs
  seite.md                         deutsche Seite (Standardsprache, unter /)
  seite.en.md                      englische Übersetzung (unter /en/)
  assets/screenshots/              erzeugte Screenshots: name.png (deutsch), name.en.png (englisch)
  erste-schritte/  funktionen/  zusammenarbeit/  migration/  hilfe/
docs/templates/migration-guide.md  Vorlage für Versionshinweise mit Handarbeit
scripts/check_docs_coverage.py     prüft, ob jede Seite der App in beiden Sprachen dokumentiert ist
scripts/docs-screenshots/          erzeugt die Screenshots mit Playwright
```

## Was automatisch passiert

| Wann | Was | Wo |
|---|---|---|
| Jeder Pull Request | Prüfung: Jede Route der App (`frontend/src/routes/routes.tsx`) ist im Front Matter genau einer Handbuchseite je Sprache eingetragen; jede Seite hat ihre Übersetzung; jeder Screenshot existiert in beiden Sprachen und wird verwendet; `mkdocs build --strict` findet keine kaputten Links | Job `docs` in `.github/workflows/ci.yml` |
| Push auf `main` (App geändert) | Screenshots werden in beiden Sprachen neu erzeugt; haben sie sich geändert, entsteht ein Pull Request „Update screenshots“ | `.github/workflows/docs-screenshots.yml` |
| Push auf `main` (Doku geändert) | Handbuch und API-Referenz werden gebaut und auf GitHub Pages veröffentlicht | `.github/workflows/docs.yml` |
| Release | Release-Notes verlinken Handbuch und Versionshinweise; PRs mit Label `breaking-change` erscheinen unter „Manual steps needed“ | `.github/release-drafter.yml` |

## Neue Funktion dokumentieren

1. **Seite anlegen**, z. B. `docs/user/funktionen/spenden.md`, dazu die englische Übersetzung `spenden.en.md` im selben Ordner. In **beiden** im Front Matter die Routen eintragen, die sie beschreiben:

    ```markdown
    ---
    routes:
      - /donations
    ---

    # Spenden
    ```

2. **In die Navigation** in `mkdocs.yml` unter `nav:` aufnehmen (deutscher Titel), den englischen Titel unter `nav_translations` ergänzen und in `funktionen/index.md` und `funktionen/index.en.md` verlinken.
3. **Screenshot** in `scripts/docs-screenshots/capture.mjs` in der Liste `SHOTS` ergänzen und einbinden:

    ```markdown
    ![Spendenübersicht](../assets/screenshots/donations.png){ .screenshot }
    ```

    Beide Sprachfassungen verweisen auf dieselbe `donations.png`; englische Seiten bekommen automatisch `donations.en.png`. Das Skript findet Knöpfe über ihre Beschriftung in den Übersetzungsdateien der App – nutze `t('…')` mit dem i18n-Schlüssel statt fester Texte.

4. Betroffene Seiten anpassen: [Rollen](../erste-schritte/rollen.md), [Übersicht](../funktionen/dashboard.md), [Datenschutz](../datenschutz.md), [Daten exportieren](../migration/daten-mitnehmen.md).
5. Braucht das Update **Handarbeit** der Nutzer? Dann Eintrag in den [Versionshinweisen](../migration/versionshinweise.md) in beiden Sprachen nach der Vorlage `docs/templates/migration-guide.md` und Label `breaking-change` am PR.

## Lokal ausprobieren

```bash
pip install -r docs/requirements.txt
npm run docs:serve        # Vorschau auf http://127.0.0.1:8000 (Englisch unter /en/)
npm run docs:build        # wie in CI: Coverage-Prüfung + strenger Build
npm run docs:screenshots  # alle Screenshots neu erzeugen (braucht Go und Node)
```

`docs:screenshots` startet ein eigenes Backend und Frontend auf den Ports 3101 und 5273 mit einer Wegwerf-Datenbank, richtet sie mit Beispieldaten ein und fotografiert alle Seiten. Deine eigene Installation bleibt unberührt. Das passiert einmal je Sprache. Damit sich Screenshots nur ändern, wenn sich die Oberfläche ändert, ist die Uhr im Browser eingefroren, Animationen sind abgeschaltet, zufällige IDs und Zeitstempel in Tabellen werden durch feste Werte ersetzt, und eine Datei wird nur überschrieben, wenn sich Pixel über die Farbtoleranz für Kantenglättung hinaus geändert haben. `DOCS_LANGUAGES=en` erzeugt nur eine Sprache.

## Schreibstil

- **Deutsch:** Du-Form, kurze Sätze, wie in der App. **Englisch:** einfach, freundlich, direkte Anrede.
- Für Ehrenamtliche ohne IT-Kenntnisse schreiben: Was muss ich klicken, was passiert dann?
- Bezeichnungen von Knöpfen und Menüs **fett** und genau so, wie sie in der App in der jeweiligen Sprache stehen.
- Ehrlich sein: Was (noch) nicht geht, steht in einem Hinweis-Kasten.
- Keine echten Personendaten in Beispielen und Screenshots.

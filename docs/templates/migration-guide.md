# Template: migration guide for a release

Use this template when a release needs **manual steps** from users. Most releases
do not: goose migrations in `server/internal/db/migrations/` run automatically on
start, and the sync protocol stays backwards compatible. Write a guide when a
release:

- changes or removes data in a way that is not fully automatic (e.g. a field is
  split, merged or dropped and users must check the result),
- changes an export or import format users rely on,
- needs every collaboration device to update at the same time (sync protocol /
  schema version bump that older clients reject),
- changes config files, paths or settings that are not migrated automatically,
- drops support for an operating system version.

How to publish it:

1. Copy the block below to the **top** of
   [`docs/user/migration/versionshinweise.md`](../user/migration/versionshinweise.md)
   (German) and write the English version into
   [`versionshinweise.en.md`](../user/migration/versionshinweise.en.md). Remove the
   "no manual steps" box in both if it is still there.
2. Link it from the release notes.
3. Label the PR `breaking-change` so release-drafter lists it prominently.

Checklist for the developer before release:

- [ ] Migration is additive where possible (new table/column instead of changing one)
- [ ] Released migrations are never edited – add a new numbered file
- [ ] Migration tested on a copy of a database from the previous release
- [ ] Old sync segments (previous schema version) are still applied correctly
- [ ] Downgrade path documented (usually: restore backup taken before the update)

---

```markdown
## Version X.Y.Z – <kurzer Titel>

**Betrifft:** <alle / nur Zusammenarbeit / nur Windows / …>
**Aufwand:** <ca. 5 Minuten>

### Was ändert sich?

<1–3 Sätze in einfacher Sprache, warum Handarbeit nötig ist.>

### Vor dem Update

1. Datensicherung erstellen (Einstellungen → Jetzt sichern).
2. <weitere Schritte, z. B. „Alle offenen Beitrittsanfragen bestätigen“>

### Nach dem Update

1. <Schritt>
2. <Prüfen: „Unter Mitglieder müssen … angezeigt werden“>

### Zusammenarbeit

<Müssen alle Geräte gleichzeitig aktualisieren? Was passiert mit Geräten, die noch die alte Version haben?>

### Zurück zur alten Version

Ältere Version installieren und die Datensicherung von vor dem Update einspielen.
Änderungen seit dem Update gehen dabei verloren.
```

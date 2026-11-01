# Updates & database versions

## Installing an update

1. **Create a backup** ([Settings → Back up now](../funktionen/einstellungen.md#backup)).
2. **Close** Open ClubManager.
3. Download the new version from the [releases page](https://github.com/Waschndolos/open-clubmanager/releases) and **install it over the old version**. Your database file and settings are kept.
4. Start Open ClubManager.
5. Check the [release notes](versionshinweise.md) to see whether the new version needs manual steps (rare).

## What happens on first start

New versions sometimes bring features that need the database to be extended (e.g. new tables for households or events). This **database migration** runs **automatically** on start:

- You don't have to do anything.
- Existing data is not deleted, it is carried over.
- With large databases, the first start may take a few seconds longer.

```
Version 1.2 → 1.3 → 1.4
  database    step 13      step 14   (run one after another on start)
```

You can also **skip** versions: updating from 1.1 straight to 1.4 is no problem, all missing steps run one after another.

## Going back to the old version?

After a migration, an **older** program version can no longer reliably open the database. To go back:

1. Install the older version.
2. Restore the **backup from before the update**.

Changes you made after the update are lost.

## Updates with collaboration

With collaboration, all devices exchange changes in a shared format. Therefore:

- **Update all devices promptly.** A device with an **older** version does not accept changes from newer versions and asks for an update. Its own changes are not lost; they are transferred once it is updated.
- Newer versions always understand changes from older versions.
- Agree within the board who updates first. Ideally everyone within a few days.

## If something goes wrong

- If the program doesn't start after the update or shows a database error: **don't delete anything**, keep the backup at hand and report it in the [issue tracker](https://github.com/Waschndolos/open-clubmanager/issues) (without real member data).

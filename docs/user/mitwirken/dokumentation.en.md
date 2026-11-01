# Maintaining the documentation

This guide grows with every new feature. This page is for everyone contributing to Open ClubManager.

## Principle

**Every user-visible feature is documented, in both languages, in the same pull request.** CI checks this automatically.

## Structure

```
mkdocs.yml                         configuration, navigation and nav translations
docs/requirements.txt              versions of MkDocs, Material theme and i18n plugin
docs/user/                         source of this guide
  page.md                          German page (default language, served at /)
  page.en.md                       English translation (served at /en/)
  assets/screenshots/              generated screenshots: name.png (German), name.en.png (English)
docs/templates/migration-guide.md  template for release notes with manual steps
scripts/check_docs_coverage.py     checks that every page of the app is documented in both languages
scripts/docs-screenshots/          generates the screenshots with Playwright
```

## What happens automatically

| When | What | Where |
|---|---|---|
| Every pull request | Check: every route of the app (`frontend/src/routes/routes.tsx`) is listed in the front matter of exactly one page per language; every page has its translation; every screenshot is used; `mkdocs build --strict` finds no broken links | `docs` job in `.github/workflows/ci.yml` |
| Push to `main` (app changed) | Screenshots are regenerated in both languages; if they changed, an "Update screenshots" pull request is opened | `.github/workflows/docs-screenshots.yml` |
| Push to `main` (docs changed) | Guide and API reference are built and published to GitHub Pages | `.github/workflows/docs.yml` |
| Release | Release notes link the guide and the update notes; PRs labelled `breaking-change` appear under "Manual steps needed" | `.github/release-drafter.yml` |

## Documenting a new feature

1. **Create the page** in German, e.g. `docs/user/funktionen/spenden.md`, and its English translation `spenden.en.md` in the same folder. In the front matter of **both**, list the routes they describe:

    ```markdown
    ---
    routes:
      - /donations
    ---

    # Donations
    ```

2. **Add it to the navigation** in `mkdocs.yml` under `nav:` (German title), add the English title under `nav_translations`, and link it from `funktionen/index.md` and `funktionen/index.en.md`.
3. **Screenshot:** add it to the `SHOTS` list in `scripts/docs-screenshots/capture.mjs` and embed it per language:

    ```markdown
    ![Donations](../assets/screenshots/donations.png){ .screenshot }
    ```

    Both language versions reference the same `donations.png`; English pages are served `donations.en.png` automatically. The script finds buttons by their label in the app's translation files, so use `t('…')` with the i18n key instead of hard-coded text.

4. Update affected pages: [roles](../erste-schritte/rollen.md), [overview](../funktionen/dashboard.md), [data protection](../datenschutz.md), [exporting your data](../migration/daten-mitnehmen.md).
5. Does the update need **manual steps** from users? Then add an entry to the [release notes](../migration/versionshinweise.md) in both languages using `docs/templates/migration-guide.md`, and label the PR `breaking-change`.

## Trying it locally

```bash
pip install -r docs/requirements.txt
npm run docs:serve        # preview at http://127.0.0.1:8000 (English at /en/)
npm run docs:build        # as in CI: coverage check + strict build
npm run docs:screenshots  # regenerate all screenshots (needs Go and Node)
```

`docs:screenshots` starts its own backend and frontend on ports 3101 and 5273 with a throwaway database, sets it up with demo data and captures every page, once per language. Your own installation is not touched. So that screenshots only change when the UI changes, the browser clock is frozen, animations are off, random IDs and timestamps in tables are replaced by fixed values, and a file is only overwritten if more than 0.1 % of its pixels changed. `DOCS_LANGUAGES=en` captures only one language.

## Writing style

- **German:** informal "du", short sentences, like in the app. **English:** plain, friendly, second person.
- Write for volunteers without IT knowledge: what do I click, what happens then?
- Put names of buttons and menus in **bold**, exactly as they appear in the app in that language.
- Be honest: what doesn't work (yet) goes in a notice box.
- No real personal data in examples or screenshots.

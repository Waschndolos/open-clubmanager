---
name: 🚀 Feature
about: Add a new feature
title: '[Feature] <short summary>'
labels: feature
---

## ✨ Description

What feature are you adding? Why is it useful?

## 📋 Related Issue

Closes #

## ✅ Checklist

- [ ] Code works as expected
- [ ] UI is updated if needed
- [ ] All texts are localized
- [ ] Tests added or updated
- [ ] User documentation updated in `docs/user/` in German (`page.md`) and English (`page.en.md`); a new page needs `routes:` front matter, a `nav` entry and an English `nav_translations` title in `mkdocs.yml`
- [ ] Screenshot added to `scripts/docs-screenshots/capture.mjs` if the feature has a new page or dialog
- [ ] Needs manual steps when updating? → guide in `docs/user/migration/versionshinweise.md` + `.en.md` (template: `docs/templates/migration-guide.md`) and label `breaking-change`

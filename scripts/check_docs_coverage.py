"""Fails when a page of the app has no page in the user documentation.

The guide in docs/user/ is bilingual: `page.md` is German (default language),
`page.en.md` its English translation. This check makes sure that

- every page has its translation (and no translation is orphaned),
- every route in frontend/src/routes/routes.tsx is listed in the `routes:`
  front matter of exactly one page per language,
- every screenshot named in scripts/docs-screenshots/capture.mjs exists in
  every language and is used by a page.

Run it in CI so the user documentation grows with every feature:

    python3 scripts/check_docs_coverage.py
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
ROUTES_FILE = ROOT / 'frontend' / 'src' / 'routes' / 'routes.tsx'
DOCS_DIR = ROOT / 'docs' / 'user'
CAPTURE_SCRIPT = ROOT / 'scripts' / 'docs-screenshots' / 'capture.mjs'
SCREENSHOT_DIR = DOCS_DIR / 'assets' / 'screenshots'

DEFAULT_LANGUAGE = 'de'
# Every other language of the guide, as suffix of the translated pages.
TRANSLATIONS = ['en']
LANGUAGES = [DEFAULT_LANGUAGE, *TRANSLATIONS]

# Routes that are no feature of their own.
IGNORED_ROUTES = {'/', '/*'}

# Top-level routes are absolute ("/login"); routes nested under the layout
# route "/" are relative ("members", "collaboration/create").
ROUTE_PATTERN = re.compile(r'path:\s*["\']([^"\']+)["\']')
FRONT_MATTER_PATTERN = re.compile(r'\A---\n(.*?)\n---\n', re.DOTALL)
SHOT_PATTERN = re.compile(r"^\s*\['([a-z0-9-]+)',\s*'/", re.MULTILINE)


def language_of(page: Path) -> str:
    suffixes = page.suffixes
    if len(suffixes) >= 2 and suffixes[-2][1:] in TRANSLATIONS:
        return suffixes[-2][1:]
    return DEFAULT_LANGUAGE


def base_name(page: Path) -> Path:
    """The page path without language suffix, e.g. funktionen/mitglieder."""
    relative = page.relative_to(DOCS_DIR)
    name = relative.name.removesuffix('.md')
    for lang in TRANSLATIONS:
        name = name.removesuffix(f'.{lang}')
    return relative.with_name(name)


def pages_by_language() -> dict[str, list[Path]]:
    pages: dict[str, list[Path]] = {lang: [] for lang in LANGUAGES}
    for page in sorted(DOCS_DIR.rglob('*.md')):
        pages[language_of(page)].append(page)
    return pages


def app_routes() -> set[str]:
    paths = ROUTE_PATTERN.findall(ROUTES_FILE.read_text(encoding='utf-8'))
    routes = {p if p.startswith('/') else f'/{p}' for p in paths}
    return routes - IGNORED_ROUTES


def documented_routes(pages: list[Path]) -> dict[str, list[Path]]:
    """Maps each route to the doc pages that claim it in their front matter."""
    claimed: dict[str, list[Path]] = {}
    for page in pages:
        match = FRONT_MATTER_PATTERN.match(page.read_text(encoding='utf-8'))
        if not match:
            continue
        in_routes = False
        for line in match.group(1).splitlines():
            if line.startswith('routes:'):
                in_routes = True
            elif in_routes and line.lstrip().startswith('- '):
                route = line.split('- ', 1)[1].strip().strip('"\'')
                claimed.setdefault(route, []).append(page.relative_to(ROOT))
            elif not line.startswith(' '):
                in_routes = False
    return claimed


def check_translations(pages: dict[str, list[Path]]) -> list[str]:
    errors = []
    default = {base_name(p) for p in pages[DEFAULT_LANGUAGE]}
    for lang in TRANSLATIONS:
        translated = {base_name(p) for p in pages[lang]}
        for missing in sorted(default - translated):
            errors.append(f'docs/user/{missing}.md has no {lang} translation (docs/user/{missing}.{lang}.md).')
        for orphan in sorted(translated - default):
            errors.append(f'docs/user/{orphan}.{lang}.md has no {DEFAULT_LANGUAGE} page (docs/user/{orphan}.md).')
    return errors


def check_routes(lang: str, pages: list[Path], routes: set[str]) -> list[str]:
    errors = []
    claimed = documented_routes(pages)
    for route in sorted(routes - claimed.keys()):
        errors.append(
            f'[{lang}] Route {route} has no documentation page. Add it to the `routes:` front matter '
            f'of the page in docs/user/ that describes it (or create one and add it to mkdocs.yml).'
        )
    for route, claiming in sorted(claimed.items()):
        if route not in routes:
            errors.append(f'[{lang}] {claiming[0]} documents route {route}, which no longer exists in the app.')
        if len(claiming) > 1:
            errors.append(f'[{lang}] Route {route} is claimed by several pages: {", ".join(map(str, claiming))}.')
    return errors


def check_screenshots(pages: list[Path]) -> list[str]:
    """Every captured screenshot exists in every language and is used by a page.

    Pages reference `screenshots/<name>.png`; the i18n plugin serves
    `<name>.<lang>.png` for translated pages.
    """
    shots = set(SHOT_PATTERN.findall(CAPTURE_SCRIPT.read_text(encoding='utf-8')))
    shots |= {p.name.split('.')[0] for p in SCREENSHOT_DIR.glob('*.png')}
    text = '\n'.join(p.read_text(encoding='utf-8') for p in pages)
    errors = []
    for shot in sorted(shots):
        if f'screenshots/{shot}.png' not in text:
            errors.append(f'Screenshot {shot}.png is captured but not used in any page of docs/user/.')
        for lang in LANGUAGES:
            suffix = '' if lang == DEFAULT_LANGUAGE else f'.{lang}'
            if not (SCREENSHOT_DIR / f'{shot}{suffix}.png').exists():
                errors.append(f'Screenshot {shot}{suffix}.png is missing; run `npm run docs:screenshots`.')
    return errors


def main() -> int:
    routes = app_routes()
    pages = pages_by_language()

    errors = check_translations(pages)
    for lang in LANGUAGES:
        errors += check_routes(lang, pages[lang], routes)
    errors += check_screenshots([p for lang in LANGUAGES for p in pages[lang]])

    if errors:
        print('User documentation is incomplete:\n')
        print('\n'.join(f'  ✗ {e}' for e in errors))
        return 1
    print(f'User documentation covers all {len(routes)} routes of the app in {", ".join(LANGUAGES)}.')
    return 0


if __name__ == '__main__':
    sys.exit(main())

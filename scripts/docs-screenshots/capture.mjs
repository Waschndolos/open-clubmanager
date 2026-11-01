// Captures the screenshots of the user documentation in one language. run.sh calls
// it once per language. German (the default language of the guide) is written to
// docs/user/assets/screenshots/<name>.png, other languages to <name>.<lang>.png;
// the MkDocs i18n plugin picks the localized file for translated pages.
//
// Needs a running frontend + backend with an EMPTY database (it walks through the
// first-run setup and imports the demo data) or a database that was set up by an
// earlier run of this script. Start them isolated, never against your real config:
//
//   SERVER_CONFIG_FILE=/tmp/ocm-docs/config.json DATABASE_PATH=/tmp/ocm-docs/club.db \
//     PORT=3101 go run ./cmd/server                       (in server/)
//   VITE_BACKEND_ORIGIN=http://localhost:3101 npx vite --port 5273   (in frontend/)
//   DOCS_LANG=en DOCS_BASE_URL=http://localhost:5273 DOCS_DB_PATH=/tmp/ocm-docs/club.db npm run capture
//
// Every shot is listed in SHOTS below; add one there when a feature gets a page.

import { chromium } from 'playwright';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import pixelmatch from 'pixelmatch';
import { PNG } from 'pngjs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const LANG = process.env.DOCS_LANG ?? 'de';
const BASE_URL = process.env.DOCS_BASE_URL ?? 'http://localhost:5273';
const API_URL = process.env.DOCS_API_URL ?? 'http://localhost:3101/api/v2';
const DB_PATH = process.env.DOCS_DB_PATH ?? path.join(here, '.data', 'clubmanager.db');
const OUT_DIR = process.env.DOCS_OUT_DIR ?? path.resolve(here, '../../docs/user/assets/screenshots');
const FILE_SUFFIX = LANG === 'de' ? '' : `.${LANG}`;
// Shown instead of the real (throwaway) database path in the screenshots.
const SHOWN_DB_PATH = LANG === 'de'
    ? 'C:\\Users\\Vorstand\\Documents\\clubmanager.db'
    : 'C:\\Users\\Board\\Documents\\clubmanager.db';
const BROWSER_LOCALE = { de: 'de-DE', en: 'en-GB', fr: 'fr-FR' }[LANG];

// Buttons and fields are found by their label in the app's own translations,
// so the script works for every language of the app.
const messages = JSON.parse(await readFile(path.resolve(here, `../../frontend/src/i18n/${LANG}.json`), 'utf-8'));
const t = (key) => key.split('.').reduce((node, part) => node[part], messages);
const escapeRegExp = (text) => text.replace(/[.*+?^$()|[\]\\{}]/g, '\\$&');
/** Matches the whole label of one or more translation keys (MUI may append " *" to required fields). */
const exact = (...keys) => new RegExp(`^(${keys.map((key) => escapeRegExp(t(key))).join('|')})\\s*\\*?$`);
// The browser clock is frozen so that dates in the screenshots (and the demo
// data, which is created relative to "today") stay the same between runs.
const FIXED_TIME = process.env.DOCS_FIXED_TIME ?? '2026-11-02T08:30:00+01:00';
// Test-only login for the throwaway docs database.
const EMAIL = process.env.DOCS_EMAIL ?? (LANG === 'de' ? 'vorstand@example.org' : 'board@example.org');
const PASSWORD = process.env.DOCS_PASSWORD ?? 'docs-screenshots';
// An existing screenshot is only replaced if more than this share of its pixels
// changed beyond pixelmatch's colour tolerance (which absorbs anti-aliasing).
// Keep it at 0: even a changed amount in a summary card is a real change.
const MIN_CHANGE = Number(process.env.DOCS_MIN_CHANGE ?? 0);

// Animations make screenshots differ from run to run.
const FREEZE_CSS = `*, *::before, *::after {
    animation: none !important; transition: none !important; caret-color: transparent !important;
}`;

/** Pages captured after login: [file name, route, optional action before the shot]. */
const SHOTS = [
    ['dashboard', '/dashboard'],
    ['search-palette', '/dashboard', async (page) => {
        await page.keyboard.press('Control+k');
        await page.getByPlaceholder(t('search.placeholder')).fill('an');
    }],
    ['members', '/members'],
    ['member-dialog', '/members', async (page) => {
        await page.getByRole('button', { name: t('members.create') }).first().click();
    }],
    ['events', '/events'],
    ['events-list', '/events', async (page) => {
        await page.getByRole('button', { name: exact('events.views.list') }).click();
    }],
    ['attendance', '/attendance'],
    ['attendance-stats', '/attendance', async (page) => {
        await page.getByRole('tab', { name: t('attendance.tabs.stats') }).click();
    }],
    ['households', '/households'],
    ['finance', '/finance'],
    ['finance-fees', '/finance', async (page) => {
        await page.getByRole('tab', { name: t('finance.tabs.memberfees') }).click();
    }],
    ['finance-open-items', '/finance', async (page) => {
        await page.getByRole('tab', { name: t('finance.tabs.openItems') }).click();
    }],
    ['finance-fee-models', '/finance', async (page) => {
        await page.getByRole('tab', { name: t('finance.tabs.feeModels') }).click();
        await page.waitForLoadState('networkidle');
    }],
    // The demo data already charged the current year, so the preview shows next year.
    ['finance-fee-run', '/finance', async (page) => {
        await page.getByRole('tab', { name: t('finance.tabs.feeRuns') }).click();
        await page.getByLabel(t('feeRuns.from')).fill('2027-01');
        await page.getByLabel(t('feeRuns.to')).fill('2027-12');
        await page.getByLabel(t('feeRuns.dueDate')).fill('2027-03-01');
        await page.getByRole('button', { name: t('feeRuns.preview') }).click();
        await page.waitForLoadState('networkidle');
        await page.getByRole('button', { name: t('feeRuns.create') }).waitFor();
    }],
    ['finance-direct-debit', '/finance', async (page) => {
        await page.getByRole('tab', { name: t('finance.tabs.directDebit') }).click();
    }],
    ['inventory', '/inventory'],
    ['honors', '/honors'],
    ['qualifications', '/qualifications'],
    ['tasks', '/tasks'],
    ['entities', '/entities'],
    ['history', '/history'],
    ['users', '/users'],
    ['collaboration', '/collaboration'],
    ['collaboration-create', '/collaboration/create'],
    ['collaboration-join', '/collaboration/join'],
    ['settings', '/settings'],
];

/** Replaces the displayed database path; only the DOM changes, not the app state. */
async function maskDbPath(page) {
    await page.evaluate(({ real, shown }) => {
        for (const input of document.querySelectorAll('input')) {
            if (input.value.includes(real)) input.value = shown;
        }
    }, { real: path.basename(path.dirname(DB_PATH)), shown: SHOWN_DB_PATH });
}

/**
 * Replaces values the server makes up on every run (UUIDs, timestamps of the
 * history) with stable ones, so that unchanged pages give identical screenshots.
 */
async function stabilizeTables(page) {
    await page.evaluate(() => {
        const cells = (field) => document.querySelectorAll(`.MuiDataGrid-cell[data-field="${field}"]`);
        ['id', 'entityId'].forEach((field) => cells(field).forEach((cell, row) => {
            cell.textContent = `01a1d0c5-0000-7000-8000-${String(row + 1).padStart(12, '0')}`;
        }));
        cells('createdAt').forEach((cell) => { cell.textContent = new Date().toLocaleString(document.documentElement.lang || undefined); });
    });
}

/** Writes the screenshot unless it differs from the existing file only by noise. */
async function save(buffer, file) {
    let existing;
    try {
        existing = PNG.sync.read(await readFile(file));
    } catch {
        existing = null;
    }
    const fresh = PNG.sync.read(buffer);
    if (existing && existing.width === fresh.width && existing.height === fresh.height) {
        const changed = pixelmatch(existing.data, fresh.data, null, fresh.width, fresh.height, { threshold: 0.1 });
        if (changed / (fresh.width * fresh.height) <= MIN_CHANGE) return 'unchanged';
    }
    await writeFile(file, buffer);
    return existing ? 'updated' : 'new';
}

async function shot(page, name) {
    await page.waitForLoadState('networkidle');
    await page.addStyleTag({ content: FREEZE_CSS });
    await page.waitForTimeout(1500); // let charts settle
    await maskDbPath(page);
    await stabilizeTables(page);
    const file = `${name}${FILE_SUFFIX}.png`;
    const result = await save(await page.screenshot(), path.join(OUT_DIR, file));
    console.log(`  ✓ ${file} (${result})`);
}

// The login token lives only in memory, so a full page load would log out again:
// navigate inside the single-page app instead.
async function navigate(page, route) {
    await page.evaluate((to) => {
        window.history.pushState({}, '', to);
        window.dispatchEvent(new PopStateEvent('popstate'));
    }, route);
}

async function runSetup(page) {
    await page.goto(`${BASE_URL}/setup`);
    await page.getByText(t('collaboration.setup.local')).first().waitFor();
    await shot(page, 'setup-usage');
    await page.getByRole('button', { name: exact('buttons.next') }).click();

    const dbField = page.getByLabel(t('setup.database.localPathLabel'));
    await dbField.fill(DB_PATH);
    await dbField.blur();
    await shot(page, 'setup-database');
    await page.getByRole('button', { name: exact('setup.database.saveButton', 'setup.database.continueButton') }).last().click();

    await page.getByLabel(exact('setup.email')).fill(EMAIL);
    await page.getByLabel(exact('setup.password')).fill(PASSWORD);
    await page.getByLabel(exact('setup.confirmPassword')).fill(PASSWORD);
    await page.getByLabel(t('setup.demoData.label')).check();
    await shot(page, 'setup-admin');
    await page.getByRole('button', { name: t('setup.createButton') }).click();
    await page.waitForURL(/\/login/, { timeout: 120_000 });
}

async function login(page) {
    await page.goto(`${BASE_URL}/login`);
    await page.getByLabel(exact('login.email')).waitFor();
    await page.waitForTimeout(4000); // login page animation
    await shot(page, 'login');
    await page.getByLabel(exact('login.email')).fill(EMAIL);
    await page.getByLabel(exact('login.password')).fill(PASSWORD);
    await page.getByRole('button', { name: exact('login.loginButton') }).click();
    await page.waitForURL(/\/dashboard/);
}

async function main() {
    await mkdir(OUT_DIR, { recursive: true });
    const browser = await chromium.launch();
    const context = await browser.newContext({
        viewport: { width: 1440, height: 900 },
        locale: BROWSER_LOCALE,
        timezoneId: 'Europe/Berlin',
        colorScheme: 'light',
        reducedMotion: 'reduce',
    });
    await context.clock.setFixedTime(new Date(FIXED_TIME));
    // The browser build has no desktop preference store, so the app falls back to
    // German; provide one that answers the language with DOCS_LANG.
    await context.addInitScript((lang) => {
        const key = (userId, name) => `userpreference:${userId}:${name}`;
        window.userpreference = {
            get(userId, name) {
                if (name === 'language') return lang;
                const value = localStorage.getItem(key(userId, name));
                try { return JSON.parse(value ?? 'null'); } catch { return value; }
            },
            set(userId, name, value) {
                if (name !== 'language') localStorage.setItem(key(userId, name), JSON.stringify(value));
            },
        };
    }, LANG);
    const page = await context.newPage();

    const status = await (await fetch(`${API_URL}/setup/status`)).json();
    if (status.setupRequired) {
        console.log(`Setup (${LANG}) …`);
        await runSetup(page);
    }
    console.log('Login …');
    await login(page);

    console.log('Pages …');
    let failed = 0;
    for (const [name, route, action] of SHOTS) {
        try {
            await navigate(page, route);
            await page.waitForLoadState('networkidle');
            if (action) await action(page);
            await shot(page, name);
        } catch (err) {
            failed++;
            console.error(`  ✗ ${name}: ${err.message.split('\n')[0]}`);
        }
        await page.keyboard.press('Escape').catch(() => {});
        await page.waitForTimeout(300);
    }
    await browser.close();
    if (failed) process.exitCode = 1;
}

main().catch((err) => {
    console.error(err);
    process.exit(1);
});

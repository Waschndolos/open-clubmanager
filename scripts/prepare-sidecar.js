#!/usr/bin/env node
/**
 * Copies the locally-built Go server binary into the src-tauri/binaries/
 * directory with the Tauri-expected name:
 *   open-clubmanager-server-<rust-target-triple>[.exe]
 *
 * Run this after `npm run build:backend` and before `tauri build`.
 * The script is invoked automatically by `npm run build`.
 */

import { execSync } from 'child_process';
import { cpSync, mkdirSync, existsSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = join(__dirname, '..');

// Resolve the current Rust target triple (e.g. x86_64-unknown-linux-gnu)
let rustcOutput;
try {
    rustcOutput = execSync('rustc -Vv', { encoding: 'utf8' });
} catch {
    console.error('Error: `rustc` not found. Install Rust via https://rustup.rs');
    process.exit(1);
}

const targetMatch = rustcOutput.match(/host:\s+(\S+)/);
if (!targetMatch) {
    console.error('Error: could not determine Rust target triple from `rustc -Vv` output.');
    process.exit(1);
}
const target = targetMatch[1];

const isWindows = target.includes('windows');
const ext = isWindows ? '.exe' : '';

const src = join(root, 'server', 'dist', `open-clubmanager-server${ext}`);
const destDir = join(root, 'src-tauri', 'binaries');
const dest = join(destDir, `open-clubmanager-server-${target}${ext}`);

if (!existsSync(src)) {
    console.error(`Error: source binary not found at ${src}`);
    console.error('Run "npm run build:backend" first.');
    process.exit(1);
}

mkdirSync(destDir, { recursive: true });
cpSync(src, dest);
console.log(`Sidecar binary ready: ${dest}`);

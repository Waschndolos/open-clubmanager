#!/usr/bin/env node

import { spawnSync } from 'child_process';
import { dirname, join } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const projectRoot = join(__dirname, '..');
const tauriDir = join(projectRoot, 'src-tauri');

function run(command, args, options = {}) {
  return spawnSync(command, args, {
    encoding: 'utf8',
    stdio: 'pipe',
    ...options,
  });
}

function fail(message, details = []) {
  console.error(`Error: ${message}`);

  for (const detail of details) {
    console.error(detail);
  }

  process.exit(1);
}

function commandExists(command) {
  const result = run(command, ['--version']);
  return result.status === 0;
}

if (!commandExists('cargo') || !commandExists('rustc')) {
  const installHint = process.platform === 'win32'
    ? 'Install Rust via https://rustup.rs and restart your terminal / IDE afterwards.'
    : [
        'Install Rust via rustup and restart your terminal / IDE afterwards:',
        '  curl --proto "=https" --tlsv1.2 -sSf https://sh.rustup.rs | sh',
        'If Rust is already installed, ensure your PATH contains ~/.cargo/bin.',
      ].join('\n');

  fail(
    'Tauri requires both `cargo` and `rustc`, but they are not available in the current PATH.',
    [installHint],
  );
}

const metadataResult = run('cargo', ['metadata', '--no-deps', '--format-version', '1'], {
  cwd: tauriDir,
});

if (metadataResult.status !== 0) {
  fail(
    'Cargo is installed, but `cargo metadata` failed for `src-tauri`.',
    [metadataResult.stderr.trim() || metadataResult.stdout.trim()],
  );
}

console.log('Tauri prerequisites check passed.');


/**
 * Returns true when running inside the Electron desktop app with Folder Mode
 * IPC available (i.e. window.api is exposed by the preload script).
 */
export function isElectronFolderMode(): boolean {
    return typeof window !== 'undefined' && 'api' in window && window.api !== undefined;
}

/**
 * Synthetic access token used in Folder Mode to satisfy the RequireAuth guard
 * without a real backend JWT. Any non-empty string works; using a constant
 * makes the intent explicit and avoids accidental typos.
 */
export const FOLDER_MODE_TOKEN = 'folder-mode';

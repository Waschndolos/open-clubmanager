/**
 * Returns true when running inside the Electron desktop app with Folder Mode
 * IPC available (i.e. window.api is exposed by the preload script).
 */
export function isElectronFolderMode(): boolean {
    return typeof window !== 'undefined' && 'api' in window && window.api !== undefined;
}

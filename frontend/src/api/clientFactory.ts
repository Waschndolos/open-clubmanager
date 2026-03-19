import { DataClient } from './dataClient';
import { ipcClient } from './ipcClient';
import { httpApiClient } from './httpApiClient';
import { isElectronFolderMode } from '../lib/environment';

/**
 * Returns the appropriate DataClient implementation:
 * - IpcClient when running inside Electron with the Folder Mode API exposed.
 * - HttpApiClient when running in a regular browser (dev:browser mode).
 */
export function getDataClient(): DataClient {
    if (isElectronFolderMode()) {
        return ipcClient;
    }
    return httpApiClient;
}

import api from './api';
import { BACKEND_URL } from './baseUrl';

const SYSTEM_URL = `${BACKEND_URL}/system`;

/**
 * Triggers a backup download of the current SQLite database.
 * The server returns a ZIP file which is saved to the user's downloads folder.
 */
export async function createBackup(): Promise<void> {
    const response = await api.post(`${SYSTEM_URL}/backup`, {}, {
        responseType: 'blob',
    });

    const blob = new Blob([response.data], { type: 'application/zip' });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    const date = new Date().toISOString().split('T')[0];

    anchor.href = url;
    anchor.download = `clubmanager-backup-${date}.zip`;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(url);
}

/**
 * Uploads a ZIP backup file to the server and restores the database from it.
 */
export async function restoreBackup(file: File): Promise<void> {
    const formData = new FormData();
    formData.append('backup', file);

    await api.post(`${SYSTEM_URL}/restore`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
    });
}

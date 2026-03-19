import * as fs from 'fs';
import * as path from 'path';
import archiver from 'archiver';

const BACKUPS_DIR = 'backups';

/**
 * Creates a ZIP backup of `club.db` and the `attachments/` folder.
 *
 * The archive is placed in `backups/<YYYY-MM-DD>_openclubmanager.zip`
 * inside the data directory.
 *
 * Returns the absolute path to the created ZIP file.
 */
export async function createBackup(dataDir: string): Promise<string> {
    const backupsDir = path.join(dataDir, BACKUPS_DIR);
    fs.mkdirSync(backupsDir, { recursive: true });

    const today = new Date().toISOString().slice(0, 10);
    const zipName = `${today}_openclubmanager.zip`;
    const zipPath = path.join(backupsDir, zipName);

    return new Promise((resolve, reject) => {
        const output = fs.createWriteStream(zipPath);
        const archive = archiver('zip', { zlib: { level: 6 } });

        output.on('close', () => resolve(zipPath));
        archive.on('error', reject);
        archive.pipe(output);

        // Include the SQLite database.
        const dbPath = path.join(dataDir, 'club.db');
        if (fs.existsSync(dbPath)) {
            archive.file(dbPath, { name: 'club.db' });
        }

        // Include the entire attachments directory (skip .tmp subdir).
        const attachmentsDir = path.join(dataDir, 'attachments');
        if (fs.existsSync(attachmentsDir)) {
            archive.glob('**/*', {
                cwd: attachmentsDir,
                ignore: ['.tmp/**'],
            }, { prefix: 'attachments' });
        }

        archive.finalize();
    });
}

import * as fs from 'fs';
import * as path from 'path';
import * as crypto from 'crypto';

const ATTACHMENTS_DIR = 'attachments';
const TMP_DIR = '.tmp';
const MAX_FILENAME_LENGTH = 200;

export interface AttachmentWriteResult {
    storedRelPath: string;
    sha256: string;
    sizeBytes: number;
}

/**
 * Handles atomic attachment file writes.
 *
 * Write strategy:
 *  1. Copy source file to `attachments/.tmp/<uuid>`
 *  2. Compute sha256 hash
 *  3. Rename into `attachments/YYYY/MM/<uuid>-<sanitizedName>`
 *     (same volume → atomic on most file systems including SMB)
 *
 * This ensures no partial files ever appear in the final location.
 */
export class AttachmentService {
    private readonly attachmentsDir: string;
    private readonly tmpDir: string;

    constructor(dataDir: string) {
        this.attachmentsDir = path.join(dataDir, ATTACHMENTS_DIR);
        this.tmpDir = path.join(this.attachmentsDir, TMP_DIR);
    }

    ensureDirs(): void {
        fs.mkdirSync(this.attachmentsDir, { recursive: true });
        fs.mkdirSync(this.tmpDir, { recursive: true });
    }

    /**
     * Atomically writes a file from `sourcePath` into the attachment store.
     * Returns the relative path (relative to `dataDir`), sha256 hash, and
     * file size in bytes.
     *
     * If the rename fails (cross-device on some unusual setups) the temp file
     * is cleaned up and an error is thrown.
     */
    async writeAttachment(
        sourcePath: string,
        originalName: string,
        uuid: string
    ): Promise<AttachmentWriteResult> {
        this.ensureDirs();

        const tmpFile = path.join(this.tmpDir, uuid);
        const sanitizedName = sanitizeFileName(originalName);

        // Step 1: Copy to temp location.
        fs.copyFileSync(sourcePath, tmpFile);

        let sha256: string;
        let sizeBytes: number;

        try {
            // Step 2: Compute hash and size on the temp file.
            ({ sha256, sizeBytes } = await computeHashAndSize(tmpFile));

            // Step 3: Build final path and ensure the date directory exists.
            const now = new Date();
            const year = String(now.getFullYear());
            const month = String(now.getMonth() + 1).padStart(2, '0');
            const dateDir = path.join(this.attachmentsDir, year, month);
            fs.mkdirSync(dateDir, { recursive: true });

            const finalName = `${uuid}-${sanitizedName}`;
            const finalPath = path.join(dateDir, finalName);
            const storedRelPath = path.join(ATTACHMENTS_DIR, year, month, finalName);

            // Atomic rename (same volume guarantees atomicity).
            fs.renameSync(tmpFile, finalPath);

            return { storedRelPath, sha256, sizeBytes };
        } catch (err) {
            // Clean up temp file on any failure to avoid orphaned temp files.
            try { fs.unlinkSync(tmpFile); } catch (cleanupErr) {
                console.error('Failed to clean up temp attachment file:', cleanupErr);
            }
            throw err;
        }
    }

    /**
     * Returns the absolute path for a given relative stored path.
     */
    resolvePath(dataDir: string, storedRelPath: string): string {
        return path.join(dataDir, storedRelPath);
    }
}

function sanitizeFileName(name: string): string {
    // Keep only safe characters; replace everything else with underscore.
    return name.replace(/[^a-zA-Z0-9._-]/g, '_').slice(0, MAX_FILENAME_LENGTH);
}

function computeHashAndSize(filePath: string): Promise<{ sha256: string; sizeBytes: number }> {
    return new Promise((resolve, reject) => {
        const hash = crypto.createHash('sha256');
        const stream = fs.createReadStream(filePath);
        let sizeBytes = 0;

        stream.on('data', (chunk: string | Buffer) => {
            const bytes = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
            sizeBytes += bytes.length;
            hash.update(bytes);
        });
        stream.on('end', () => resolve({ sha256: hash.digest('hex'), sizeBytes }));
        stream.on('error', reject);
    });
}

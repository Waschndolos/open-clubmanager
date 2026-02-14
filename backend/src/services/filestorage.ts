import { promises as fs } from 'fs';
import { join } from 'path';
import { encrypt, decrypt, getKeyFromPassphrase } from '../utils/crypto';

interface EncryptedPayload {
    iv: string;
    authTag: string;
    data: string;
}

export class FileStorage<T = unknown> {
    private readonly dir: string;
    private readonly key: Buffer;

    constructor(storageDir: string) {
        this.dir = storageDir;
        const pass = process.env.ENCRYPTION_KEY || "dummy";
        if (!pass) {
            throw new Error('ENCRYPTION_KEY env variable not set');
        }
        this.key = getKeyFromPassphrase(pass);
    }

    private filePath(id: string): string {
        return join(this.dir, `${id}.json`);
    }

    async write(id: string, data: T): Promise<void> {
        const json = JSON.stringify(data, null, 2);
        const payload: EncryptedPayload = encrypt(json, this.key);
        await fs.mkdir(this.dir, { recursive: true });
        await fs.writeFile(this.filePath(id), JSON.stringify(payload, null, 2), 'utf8');
    }

    async read(id: string): Promise<T | null> {
        try {
            const raw = await fs.readFile(this.filePath(id), 'utf8');
            const payload: EncryptedPayload = JSON.parse(raw);
            const decrypted = decrypt(payload, this.key);
            return JSON.parse(decrypted) as T;
        } catch (err) {
            if ((err as NodeJS.ErrnoException).code === 'ENOENT') return null;
            throw err;
        }
    }

    async list(): Promise<string[]> {
        try {
            const files = await fs.readdir(this.dir);
            return files
                .filter((f) => f.endsWith('.json'))
                .map((f) => f.replace(/\.json$/, ''));
        } catch (err) {
            if ((err as NodeJS.ErrnoException).code === 'ENOENT') return [];
            throw err;
        }
    }

    async delete(id: string): Promise<void> {
        try {
            await fs.unlink(this.filePath(id));
        } catch (err) {
            if ((err as NodeJS.ErrnoException).code !== 'ENOENT') throw err;
        }
    }
}
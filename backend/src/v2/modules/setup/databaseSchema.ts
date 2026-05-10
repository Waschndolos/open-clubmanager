import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';
import { HttpError } from '../../core/httpError.ts';
import { DatabaseMode } from '../preferences/appConfigStore.ts';

const execFileAsync = promisify(execFile);
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const backendRoot = path.join(__dirname, '..', '..', '..', '..');

export async function ensureDatabaseSchema(mode: DatabaseMode, databaseUrl: string): Promise<void> {
    const schemaPath = mode === 'sqlite-local'
        ? 'prisma/schema.sqlite.prisma'
        : 'prisma/schema.mysql.prisma';

    try {
        await execFileAsync(
            'npx',
            ['prisma', 'db', 'push', '--schema', schemaPath],
            {
                cwd: backendRoot,
                env: {
                    ...process.env,
                    DATABASE_URL: databaseUrl,
                },
            }
        );
    } catch (error) {
        const stderr = (error as { stderr?: string }).stderr;
        throw new HttpError(
            500,
            'SETUP_DATABASE_SCHEMA_FAILED',
            stderr
                ? `Failed to prepare database schema: ${stderr}`
                : 'Failed to prepare database schema.'
        );
    }
}


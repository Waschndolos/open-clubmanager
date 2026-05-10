import fs from 'fs/promises';
import { fileURLToPath } from 'url';
import * as path from 'path';
import { HttpError } from '../../core/httpError.ts';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const configFilePath = path.join(__dirname, '..', '..', '..', '..', 'app-settings.json');

export type DatabaseMode = 'sqlite-local' | 'mysql-shared';

export interface AppConfig {
    DATABASE_URL: string;
    DATABASE_MODE: DatabaseMode;
}

const DEFAULT_APP_CONFIG: AppConfig = {
    DATABASE_URL: '',
    DATABASE_MODE: 'sqlite-local',
};

export async function getAppConfig(): Promise<AppConfig> {
    try {
        const configContent = await fs.readFile(configFilePath, 'utf8');
        const parsed = JSON.parse(configContent) as Partial<AppConfig>;
        return {
            DATABASE_URL: typeof parsed.DATABASE_URL === 'string' ? parsed.DATABASE_URL : '',
            DATABASE_MODE: parsed.DATABASE_MODE === 'mysql-shared' ? 'mysql-shared' : 'sqlite-local',
        };
    } catch (err) {
        const errorCode = (err as NodeJS.ErrnoException).code;
        if (errorCode !== 'ENOENT') {
            throw err;
        }

        await setAppConfig(DEFAULT_APP_CONFIG);
        return { ...DEFAULT_APP_CONFIG };
    }
}

export async function setAppConfig(config: AppConfig): Promise<void> {
    await fs.writeFile(configFilePath, JSON.stringify(config, null, 2), 'utf8');
}

export function isDatabaseUrlValid(mode: DatabaseMode, databaseUrl: string): boolean {
    if (!databaseUrl) {
        return false;
    }

    if (mode === 'sqlite-local') {
        return databaseUrl.startsWith('file:');
    }

    return /^mysqls?:\/\//i.test(databaseUrl);
}

export function validateDatabaseConfiguration(mode: DatabaseMode, databaseUrl: string): void {
    if (!isDatabaseUrlValid(mode, databaseUrl)) {
        throw new HttpError(
            400,
            'SETUP_INVALID_DATABASE_URL',
            mode === 'sqlite-local'
                ? 'For local mode, database URL must start with "file:".'
                : 'For shared mode, database URL must start with "mysql://" or "mysqls://".'
        );
    }
}


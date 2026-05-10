import { SetupRepository } from './setupRepository.ts';
import { HttpError } from '../../core/httpError.ts';
import bcrypt from 'bcrypt';
import {
    DatabaseMode,
    getAppConfig,
    isDatabaseUrlValid,
    setAppConfig,
    validateDatabaseConfiguration,
} from '../preferences/appConfigStore.ts';
import { setActiveDatabase } from '../../../db.ts';
import { ensureDatabaseSchema } from './databaseSchema.ts';

export interface SetupStatus {
    setupRequired: boolean;
    userCount: number;
    databaseConfigured: boolean;
    databaseMode: DatabaseMode;
}

export interface SetupDatabaseConfiguration {
    mode: DatabaseMode;
    databaseUrl: string;
}

export class SetupService {
    constructor(private readonly setupRepository: SetupRepository) {}

    async getStatus(): Promise<SetupStatus> {
        const config = await getAppConfig();
        const databaseConfigured = isDatabaseUrlValid(config.DATABASE_MODE, config.DATABASE_URL);

        if (!databaseConfigured) {
            return {
                setupRequired: true,
                userCount: 0,
                databaseConfigured: false,
                databaseMode: config.DATABASE_MODE,
            };
        }

        try {
            await setActiveDatabase(config.DATABASE_MODE, config.DATABASE_URL);

            const userCount = await this.setupRepository.countUsers();

            return {
                setupRequired: userCount === 0,
                userCount,
                databaseConfigured: true,
                databaseMode: config.DATABASE_MODE,
            };
        } catch {
            return {
                setupRequired: true,
                userCount: 0,
                databaseConfigured: true,
                databaseMode: config.DATABASE_MODE,
            };
        }
    }

    async configureDatabase(mode: DatabaseMode, databaseUrl: string | undefined): Promise<SetupDatabaseConfiguration> {
        const normalizedUrl = (databaseUrl ?? '').trim();
        const currentConfig = await getAppConfig();
        const envDatabaseUrl = (process.env.DATABASE_URL ?? '').trim();

        const resolvedUrl = mode === 'sqlite-local'
            ? (
                normalizedUrl
                || (isDatabaseUrlValid('sqlite-local', currentConfig.DATABASE_URL) ? currentConfig.DATABASE_URL : '')
                || (isDatabaseUrlValid('sqlite-local', envDatabaseUrl) ? envDatabaseUrl : '')
                || 'file:./clubmanager.db'
            )
            : normalizedUrl;

        validateDatabaseConfiguration(mode, resolvedUrl);

        await ensureDatabaseSchema(mode, resolvedUrl);

        await setAppConfig({
            DATABASE_MODE: mode,
            DATABASE_URL: resolvedUrl,
        });

        await setActiveDatabase(mode, resolvedUrl);

        return {
            mode,
            databaseUrl: resolvedUrl,
        };
    }

    async initializeAdmin(email: string, password: string): Promise<void> {
        const config = await getAppConfig();
        if (!isDatabaseUrlValid(config.DATABASE_MODE, config.DATABASE_URL)) {
            throw new HttpError(400, 'SETUP_DATABASE_NOT_CONFIGURED', 'Please configure the database before creating the admin user.');
        }

        await setActiveDatabase(config.DATABASE_MODE, config.DATABASE_URL);

        const normalizedEmail = email.trim().toLowerCase();

        if (!normalizedEmail || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizedEmail)) {
            throw new HttpError(400, 'SETUP_INVALID_EMAIL', 'A valid email address is required.');
        }

        if (!password || password.length < 8) {
            throw new HttpError(400, 'SETUP_INVALID_PASSWORD', 'Password must be at least 8 characters long.');
        }

        let userCount = 0;
        try {
            userCount = await this.setupRepository.countUsers();
        } catch {
            throw new HttpError(
                500,
                'SETUP_DATABASE_CONNECTION_FAILED',
                config.DATABASE_MODE === 'mysql-shared'
                    ? 'Could not connect to the configured MySQL database. Please verify URL and backend DB support.'
                    : 'Could not connect to the configured SQLite database.'
            );
        }

        if (userCount > 0) {
            throw new HttpError(403, 'SETUP_ALREADY_COMPLETED', 'Setup has already been completed.');
        }

        const passwordHash = await bcrypt.hash(password, 10);
        await this.setupRepository.createInitialAdmin(normalizedEmail, passwordHash);
    }
}


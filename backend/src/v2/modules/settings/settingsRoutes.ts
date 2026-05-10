import { Router } from 'express';
import { getCurrentDbPath, setActiveDatabase } from '../../../db.ts';
import { asyncHandler } from '../../core/asyncHandler.ts';
import {
    DatabaseMode,
    getAppConfig,
    setAppConfig,
    validateDatabaseConfiguration,
} from '../preferences/appConfigStore.ts';
import { ensureDatabaseSchema } from '../setup/databaseSchema.ts';

export function createSettingsRoutes(): Router {
    const router = Router();

    router.get('/database', asyncHandler(async (_req, res) => {
        const config = await getAppConfig();
        res.json({
            mode: config.DATABASE_MODE,
            databaseUrl: config.DATABASE_URL,
        });
    }));

    router.post('/database', asyncHandler(async (req, res) => {
        const { mode, databaseUrl } = req.body as { mode?: DatabaseMode; databaseUrl?: string };

        if (mode !== 'sqlite-local' && mode !== 'mysql-shared') {
            res.status(400).json({ error: 'Invalid database mode.' });
            return;
        }

        const normalizedUrl = (databaseUrl ?? '').trim();
        validateDatabaseConfiguration(mode, normalizedUrl);
        await ensureDatabaseSchema(mode, normalizedUrl);

        await setAppConfig({
            DATABASE_MODE: mode,
            DATABASE_URL: normalizedUrl,
        });
        await setActiveDatabase(mode, normalizedUrl);

        res.json({ status: 'ok', mode, databaseUrl: normalizedUrl });
    }));

    router.post('/set-db-path', asyncHandler(async (req, res) => {
        const { dbPath } = req.body as { dbPath?: string };
        const normalizedUrl = dbPath?.startsWith('file:') ? dbPath : `file:${dbPath ?? ''}`;

        validateDatabaseConfiguration('sqlite-local', normalizedUrl);
        await ensureDatabaseSchema('sqlite-local', normalizedUrl);
        await setAppConfig({
            DATABASE_MODE: 'sqlite-local',
            DATABASE_URL: normalizedUrl,
        });
        await setActiveDatabase('sqlite-local', normalizedUrl);

        res.json({ status: 'ok', dbPath });
    }));

    router.get('/db-path', asyncHandler(async (_req, res) => {
        const dbPath = getCurrentDbPath();
        res.json({ dbPath });
    }));

    return router;
}


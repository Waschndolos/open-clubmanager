import { Router } from 'express';
import { asyncHandler } from '../../core/asyncHandler.ts';
import { SetupService } from './setupService.ts';
import { DatabaseMode } from '../preferences/appConfigStore.ts';

export function createSetupRoutes(setupService: SetupService): Router {
    const router = Router();

    router.get('/status', asyncHandler(async (_req, res) => {
        const status = await setupService.getStatus();
        res.json(status);
    }));

    router.post('/initialize', asyncHandler(async (req, res) => {
        const { email, password } = req.body as { email?: string; password?: string };
        await setupService.initializeAdmin(email ?? '', password ?? '');
        res.status(201).json({ success: true });
    }));

    router.post('/configure-database', asyncHandler(async (req, res) => {
        const { mode, databaseUrl } = req.body as { mode?: DatabaseMode; databaseUrl?: string };

        if (mode !== 'sqlite-local' && mode !== 'mysql-shared') {
            res.status(400).json({
                code: 'SETUP_INVALID_DATABASE_MODE',
                error: 'Database mode must be "sqlite-local" or "mysql-shared".',
            });
            return;
        }

        const result = await setupService.configureDatabase(mode, databaseUrl);
        res.status(200).json(result);
    }));

    return router;
}


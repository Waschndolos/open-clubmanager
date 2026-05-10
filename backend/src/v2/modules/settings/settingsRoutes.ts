import { Router } from 'express';
import { getCurrentDbPath, setActiveDb } from '../../../db.ts';
import { asyncHandler } from '../../core/asyncHandler.ts';

export function createSettingsRoutes(): Router {
    const router = Router();

    router.post('/set-db-path', asyncHandler(async (req, res) => {
        const { dbPath } = req.body as { dbPath?: string };
        await setActiveDb(dbPath);
        res.json({ status: 'ok', dbPath });
    }));

    router.get('/db-path', asyncHandler(async (_req, res) => {
        const dbPath = getCurrentDbPath();
        res.json({ dbPath });
    }));

    return router;
}


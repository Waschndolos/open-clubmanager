import { Router } from 'express';
import { asyncHandler } from '../../core/asyncHandler.ts';
import { SetupService } from './setupService.ts';

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

    return router;
}


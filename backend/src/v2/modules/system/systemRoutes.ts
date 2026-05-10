import { Router } from 'express';
import { ApiV2Config } from '../../config.ts';

export function createSystemRoutes(config: ApiV2Config): Router {
    const router = Router();

    router.get('/health', (_req, res) => {
        res.json({ status: 'ok', service: 'open-clubmanager-api-v2' });
    });

    router.get('/meta', (_req, res) => {
        res.json({
            apiVersion: 'v2',
            usesLegacyJwtFallback: config.usesLegacyFallback,
            now: new Date().toISOString(),
        });
    });

    return router;
}


import { Router } from 'express';
import { getClient } from '../../../db.ts';
import { ApiV2Config } from '../../config.ts';
import { createVerifyToken } from '../auth/authMiddleware.ts';
import { asyncHandler } from '../../core/asyncHandler.ts';

export function createHistoryRoutes(config: ApiV2Config): Router {
    const router = Router();
    const verifyToken = createVerifyToken(config);

    router.get('/', verifyToken, asyncHandler(async (_req, res) => {
        const prisma = await getClient();
        const logs = await prisma.auditLog.findMany({
            orderBy: { createdAt: 'desc' },
            take: 500,
        });
        res.json(logs);
    }));

    return router;
}


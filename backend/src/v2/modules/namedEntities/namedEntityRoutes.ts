import { Router } from 'express';
import { getClient } from '../../../db.ts';
import { ApiV2Config } from '../../config.ts';
import { createVerifyToken, AuthenticatedRequest } from '../auth/authMiddleware.ts';
import { asyncHandler } from '../../core/asyncHandler.ts';
import { createAuditLog } from '../history/historyAudit.ts';

type DbClient = Awaited<ReturnType<typeof getClient>>;

type ModelDelegate = {
    findMany: () => Promise<unknown[]>;
    create: (args: { data: { name: string } }) => Promise<unknown>;
    update: (args: { where: { id: number }; data: { name: string } }) => Promise<unknown>;
    delete: (args: { where: { id: number } }) => Promise<unknown>;
};

export function createNamedEntityRoutes(
    config: ApiV2Config,
    getDelegate: (prisma: DbClient) => ModelDelegate,
    entityName: string
): Router {
    const router = Router();
    const verifyToken = createVerifyToken(config);

    router.get('/', asyncHandler(async (_req, res) => {
        const prisma = await getClient();
        const items = await getDelegate(prisma).findMany();
        res.json(items);
    }));

    router.post('/', verifyToken, asyncHandler(async (req: AuthenticatedRequest, res) => {
        const { name } = req.body as { name?: string };
        const prisma = await getClient();
        const item = await getDelegate(prisma).create({ data: { name: name ?? '' } }) as { id: number };
        await createAuditLog(prisma, 'CREATE', entityName, item.id, req.userEmail ?? '', { name });
        res.status(201).json(item);
    }));

    router.put('/:id', verifyToken, asyncHandler(async (req: AuthenticatedRequest, res) => {
        const id = Number(req.params.id);
        const { name } = req.body as { name?: string };
        const prisma = await getClient();
        const item = await getDelegate(prisma).update({
            where: { id },
            data: { name: name ?? '' },
        }) as { id: number };
        await createAuditLog(prisma, 'UPDATE', entityName, item.id, req.userEmail ?? '', { name });
        res.json(item);
    }));

    router.delete('/:id', verifyToken, asyncHandler(async (req: AuthenticatedRequest, res) => {
        const id = Number(req.params.id);
        const prisma = await getClient();
        await getDelegate(prisma).delete({ where: { id } });
        await createAuditLog(prisma, 'DELETE', entityName, id, req.userEmail ?? '');
        res.sendStatus(204);
    }));

    return router;
}


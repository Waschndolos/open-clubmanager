import express from 'express';
import { getClient } from '../db.ts';
import { PrismaClient } from '../generated/prisma/client.ts';

type ModelDelegate = {
    findMany: () => Promise<unknown[]>;
    create: (args: { data: { name: string } }) => Promise<unknown>;
    update: (args: { where: { id: number }; data: { name: string } }) => Promise<unknown>;
    delete: (args: { where: { id: number } }) => Promise<unknown>;
};

export function createCrudRouter(
    getDelegate: (prisma: PrismaClient) => ModelDelegate,
    entityName: string
): express.Router {
    const router = express.Router();

    router.get('/', async (_, res) => {
        const prisma = await getClient();
        const items = await getDelegate(prisma).findMany();
        res.json(items);
    });

    router.post('/', async (req, res) => {
        const { name } = req.body;
        try {
            const prisma = await getClient();
            const item = await getDelegate(prisma).create({ data: { name } });
            res.status(201).json(item);
        } catch (_e) {
            res.status(400).json({ error: `Could not create ${entityName}` });
        }
    });

    router.put('/:id', async (req, res) => {
        const { id } = req.params;
        const { name } = req.body;
        try {
            const prisma = await getClient();
            const item = await getDelegate(prisma).update({
                where: { id: Number(id) },
                data: { name },
            });
            res.json(item);
        } catch (_e) {
            res.status(400).json({ error: `Could not update ${entityName}` });
        }
    });

    router.delete('/:id', async (req, res) => {
        const { id } = req.params;
        try {
            const prisma = await getClient();
            await getDelegate(prisma).delete({ where: { id: Number(id) } });
            res.sendStatus(204);
        } catch (_e) {
            res.status(400).json({ error: `Could not delete ${entityName}` });
        }
    });

    return router;
}

import express from 'express';
import { getClient } from '../db.ts';
import { verifyToken } from '../middlewares/auth.ts';
import { PrismaClient } from '../generated/prisma/client.ts';

const router = express.Router();

// GET /api/history – List audit log entries (most recent first)
router.get('/', verifyToken, async (_req, res) => {
    try {
        const prisma = await getClient();
        const logs = await prisma.auditLog.findMany({
            orderBy: { createdAt: 'desc' },
            take: 500,
        });
        res.json(logs);
    } catch (err) {
        console.error(err);
        res.status(500).json({ error: 'Error loading history' });
    }
});

export async function createAuditLog(
    prisma: PrismaClient,
    action: 'CREATE' | 'UPDATE' | 'DELETE',
    entity: string,
    entityId: number,
    userId: string,
    data?: object
): Promise<void> {
    if (!userId) {
        console.error('Audit log skipped: userId is empty');
        return;
    }
    try {
        await prisma.auditLog.create({
            data: {
                action,
                entity,
                entityId,
                userId,
                data: data ? JSON.stringify(data) : undefined,
            },
        });
    } catch (err) {
        console.error('Failed to write audit log:', err);
    }
}

export default router;

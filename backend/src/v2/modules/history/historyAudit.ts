import { PrismaClient } from '../../../generated/prisma/client.ts';

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


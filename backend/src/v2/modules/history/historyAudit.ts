
type AuditLogDbClient = {
    auditLog: {
        create(args: Record<string, unknown>): Promise<unknown>;
    };
};

export async function createAuditLog(
    prisma: AuditLogDbClient,
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
        const auditLogDelegate = (prisma as unknown as AuditLogDbClient).auditLog;
        await auditLogDelegate.create({
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


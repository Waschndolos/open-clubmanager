import { Router } from 'express';
import { getClient } from '../../../db.ts';
import { ApiV2Config } from '../../config.ts';
import { createVerifyToken, AuthenticatedRequest } from '../auth/authMiddleware.ts';
import { asyncHandler } from '../../core/asyncHandler.ts';
import { createAuditLog } from '../history/historyAudit.ts';

export function createFinanceRoutes(config: ApiV2Config): Router {
    const router = Router();
    const verifyToken = createVerifyToken(config);

    router.get('/transactions', verifyToken, asyncHandler(async (_req, res) => {
        const prisma = await getClient();
        const transactions = await prisma.financeTransaction.findMany({
            orderBy: { date: 'desc' },
        });
        res.json(transactions);
    }));

    router.post('/transactions', verifyToken, asyncHandler(async (req: AuthenticatedRequest, res) => {
        const { date, description, amount, type, category, notes } = req.body as Record<string, unknown>;
        if (!date || !description || amount === undefined || !type) {
            res.status(400).json({ error: 'Missing required fields' });
            return;
        }

        const prisma = await getClient();
        const transaction = await prisma.financeTransaction.create({
            data: {
                date: new Date(String(date)),
                description: String(description),
                amount: Number(amount),
                type: String(type),
                category: category ? String(category) : null,
                notes: notes ? String(notes) : null,
            },
        });

        await createAuditLog(prisma, 'CREATE', 'FinanceTransaction', transaction.id, req.userEmail ?? '', {
            description,
            amount,
            type,
        });

        res.status(201).json(transaction);
    }));

    router.put('/transactions/:id', verifyToken, asyncHandler(async (req: AuthenticatedRequest, res) => {
        const id = Number(req.params.id);
        const { date, description, amount, type, category, notes } = req.body as Record<string, unknown>;
        const prisma = await getClient();

        const transaction = await prisma.financeTransaction.update({
            where: { id },
            data: {
                date: date ? new Date(String(date)) : undefined,
                description: description ? String(description) : undefined,
                amount: amount !== undefined ? Number(amount) : undefined,
                type: type ? String(type) : undefined,
                category: category === undefined ? undefined : (category ? String(category) : null),
                notes: notes === undefined ? undefined : (notes ? String(notes) : null),
            },
        });

        await createAuditLog(prisma, 'UPDATE', 'FinanceTransaction', transaction.id, req.userEmail ?? '', {
            description,
            amount,
            type,
        });

        res.json(transaction);
    }));

    router.delete('/transactions/:id', verifyToken, asyncHandler(async (req: AuthenticatedRequest, res) => {
        const id = Number(req.params.id);
        const prisma = await getClient();
        await prisma.financeTransaction.delete({ where: { id } });
        await createAuditLog(prisma, 'DELETE', 'FinanceTransaction', id, req.userEmail ?? '');
        res.sendStatus(204);
    }));

    router.get('/memberfees', verifyToken, asyncHandler(async (_req, res) => {
        const prisma = await getClient();
        const fees = await prisma.memberFee.findMany({
            include: { member: { select: { id: true, firstName: true, lastName: true, number: true } } },
            orderBy: [{ year: 'desc' }, { dueDate: 'asc' }],
        });
        res.json(fees);
    }));

    router.post('/memberfees', verifyToken, asyncHandler(async (req: AuthenticatedRequest, res) => {
        const { memberId, amount, dueDate, paidDate, description, year } = req.body as Record<string, unknown>;

        if (!memberId || amount === undefined || !dueDate || !year) {
            res.status(400).json({ error: 'Missing required fields' });
            return;
        }

        const prisma = await getClient();
        const fee = await prisma.memberFee.create({
            data: {
                memberId: Number(memberId),
                amount: Number(amount),
                dueDate: new Date(String(dueDate)),
                paidDate: paidDate ? new Date(String(paidDate)) : null,
                description: description ? String(description) : null,
                year: Number(year),
            },
            include: { member: { select: { id: true, firstName: true, lastName: true, number: true } } },
        });

        await createAuditLog(prisma, 'CREATE', 'MemberFee', fee.id, req.userEmail ?? '', { memberId, amount, year });
        res.status(201).json(fee);
    }));

    router.put('/memberfees/:id', verifyToken, asyncHandler(async (req: AuthenticatedRequest, res) => {
        const id = Number(req.params.id);
        const { memberId, amount, dueDate, paidDate, description, year } = req.body as Record<string, unknown>;
        const prisma = await getClient();

        const fee = await prisma.memberFee.update({
            where: { id },
            data: {
                memberId: memberId !== undefined ? Number(memberId) : undefined,
                amount: amount !== undefined ? Number(amount) : undefined,
                dueDate: dueDate ? new Date(String(dueDate)) : undefined,
                paidDate: paidDate !== undefined ? (paidDate ? new Date(String(paidDate)) : null) : undefined,
                description: description !== undefined ? (description ? String(description) : null) : undefined,
                year: year !== undefined ? Number(year) : undefined,
            },
            include: { member: { select: { id: true, firstName: true, lastName: true, number: true } } },
        });

        await createAuditLog(prisma, 'UPDATE', 'MemberFee', fee.id, req.userEmail ?? '', { memberId, amount, year });
        res.json(fee);
    }));

    router.delete('/memberfees/:id', verifyToken, asyncHandler(async (req: AuthenticatedRequest, res) => {
        const id = Number(req.params.id);
        const prisma = await getClient();
        await prisma.memberFee.delete({ where: { id } });
        await createAuditLog(prisma, 'DELETE', 'MemberFee', id, req.userEmail ?? '');
        res.sendStatus(204);
    }));

    return router;
}


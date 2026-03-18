import express from 'express';
import { getClient } from '../db.ts';
import { verifyToken, AuthRequest } from '../middlewares/auth.ts';
import { createAuditLog } from './history.ts';

const router = express.Router();

// ─── Finance Transactions (Kassenbuch) ───────────────────────────────────────

router.get('/transactions', verifyToken, async (_req, res) => {
    try {
        const prisma = await getClient();
        const transactions = await prisma.financeTransaction.findMany({
            orderBy: { date: 'desc' },
        });
        res.json(transactions);
    } catch (err) {
        console.error(err);
        res.status(500).json({ error: 'Error loading transactions' });
    }
});

router.post('/transactions', verifyToken, async (req: AuthRequest, res) => {
    try {
        const { date, description, amount, type, category, notes } = req.body;
        if (!date || !description || amount === undefined || !type) {
            res.status(400).json({ error: 'Missing required fields' });
            return;
        }
        const prisma = await getClient();
        const transaction = await prisma.financeTransaction.create({
            data: { date: new Date(date), description, amount: Number(amount), type, category, notes },
        });
        await createAuditLog(prisma, 'CREATE', 'FinanceTransaction', transaction.id, req.user!, { description, amount, type });
        res.status(201).json(transaction);
    } catch (err) {
        console.error(err);
        res.status(500).json({ error: 'Error creating transaction' });
    }
});

router.put('/transactions/:id', verifyToken, async (req: AuthRequest, res) => {
    try {
        const id = parseInt(String(req.params.id));
        const { date, description, amount, type, category, notes } = req.body;
        const prisma = await getClient();
        const transaction = await prisma.financeTransaction.update({
            where: { id },
            data: {
                date: date ? new Date(date) : undefined,
                description,
                amount: amount !== undefined ? Number(amount) : undefined,
                type,
                category,
                notes,
            },
        });
        await createAuditLog(prisma, 'UPDATE', 'FinanceTransaction', transaction.id, req.user!, { description, amount, type });
        res.json(transaction);
    } catch (err) {
        console.error(err);
        res.status(500).json({ error: 'Error updating transaction' });
    }
});

router.delete('/transactions/:id', verifyToken, async (req: AuthRequest, res) => {
    try {
        const id = parseInt(String(req.params.id));
        const prisma = await getClient();
        await prisma.financeTransaction.delete({ where: { id } });
        await createAuditLog(prisma, 'DELETE', 'FinanceTransaction', id, req.user!);
        res.sendStatus(204);
    } catch (err) {
        console.error(err);
        res.status(500).json({ error: 'Error deleting transaction' });
    }
});

// ─── Member Fees ──────────────────────────────────────────────────────────────

router.get('/memberfees', verifyToken, async (_req, res) => {
    try {
        const prisma = await getClient();
        const fees = await prisma.memberFee.findMany({
            include: { member: { select: { id: true, firstName: true, lastName: true, number: true } } },
            orderBy: [{ year: 'desc' }, { dueDate: 'asc' }],
        });
        res.json(fees);
    } catch (err) {
        console.error(err);
        res.status(500).json({ error: 'Error loading member fees' });
    }
});

router.post('/memberfees', verifyToken, async (req: AuthRequest, res) => {
    try {
        const { memberId, amount, dueDate, paidDate, description, year } = req.body;
        if (!memberId || amount === undefined || !dueDate || !year) {
            res.status(400).json({ error: 'Missing required fields' });
            return;
        }
        const prisma = await getClient();
        const fee = await prisma.memberFee.create({
            data: {
                memberId: Number(memberId),
                amount: Number(amount),
                dueDate: new Date(dueDate),
                paidDate: paidDate ? new Date(paidDate) : null,
                description,
                year: Number(year),
            },
            include: { member: { select: { id: true, firstName: true, lastName: true, number: true } } },
        });
        await createAuditLog(prisma, 'CREATE', 'MemberFee', fee.id, req.user!, { memberId, amount, year });
        res.status(201).json(fee);
    } catch (err) {
        console.error(err);
        res.status(500).json({ error: 'Error creating member fee' });
    }
});

router.put('/memberfees/:id', verifyToken, async (req: AuthRequest, res) => {
    try {
        const id = parseInt(String(req.params.id));
        const { memberId, amount, dueDate, paidDate, description, year } = req.body;
        const prisma = await getClient();
        const fee = await prisma.memberFee.update({
            where: { id },
            data: {
                memberId: memberId !== undefined ? Number(memberId) : undefined,
                amount: amount !== undefined ? Number(amount) : undefined,
                dueDate: dueDate ? new Date(dueDate) : undefined,
                paidDate: paidDate !== undefined ? (paidDate ? new Date(paidDate) : null) : undefined,
                description,
                year: year !== undefined ? Number(year) : undefined,
            },
            include: { member: { select: { id: true, firstName: true, lastName: true, number: true } } },
        });
        await createAuditLog(prisma, 'UPDATE', 'MemberFee', fee.id, req.user!, { memberId, amount, year });
        res.json(fee);
    } catch (err) {
        console.error(err);
        res.status(500).json({ error: 'Error updating member fee' });
    }
});

router.delete('/memberfees/:id', verifyToken, async (req: AuthRequest, res) => {
    try {
        const id = parseInt(String(req.params.id));
        const prisma = await getClient();
        await prisma.memberFee.delete({ where: { id } });
        await createAuditLog(prisma, 'DELETE', 'MemberFee', id, req.user!);
        res.sendStatus(204);
    } catch (err) {
        console.error(err);
        res.status(500).json({ error: 'Error deleting member fee' });
    }
});

export default router;

import { Router } from 'express';
import { getClient } from '../../../db.ts';
import { ApiV2Config } from '../../config.ts';
import { createVerifyToken, AuthenticatedRequest } from '../auth/authMiddleware.ts';
import { asyncHandler } from '../../core/asyncHandler.ts';
import { createAuditLog } from '../history/historyAudit.ts';
import { parseCamt053, ParsedBankTransaction } from './camt053Import.ts';

type MemberSummary = {
    id: number;
    firstName: string;
    lastName: string;
    number: number;
    email: string;
    iban: string | null;
};

function normalizeText(value: string): string {
    return value
        .toLocaleLowerCase('de-DE')
        .normalize('NFD')
        .replace(/[\u0300-\u036f]/g, ' ')
        .replace(/[^a-z0-9]+/g, ' ')
        .trim();
}

function normalizeCompact(value: string): string {
    return value
        .toLocaleLowerCase('de-DE')
        .normalize('NFD')
        .replace(/[\u0300-\u036f]/g, '')
        .replace(/[^a-z0-9]/g, '');
}

function normalizeIban(value: string): string {
    return value.toUpperCase().replace(/[^A-Z0-9]/g, '');
}

function hasWord(text: string, word: string): boolean {
    return new RegExp(`(^|\\s)${word}(\\s|$)`).test(text);
}

function findMatchingMember(transaction: ParsedBankTransaction, members: MemberSummary[]): MemberSummary | null {
    const normalizedText = normalizeText([
        transaction.description,
        transaction.counterpartyName ?? '',
        transaction.endToEndId ?? '',
        transaction.mandateId ?? '',
        transaction.notes,
    ].join(' '));
    const compactText = normalizeCompact([
        transaction.description,
        transaction.counterpartyName ?? '',
        transaction.endToEndId ?? '',
        transaction.mandateId ?? '',
        transaction.notes,
    ].join(' '));

    if (!normalizedText && !compactText) {
        return null;
    }

    let bestMatch: { member: MemberSummary; score: number } | null = null;
    const txIban = transaction.counterpartyIban ? normalizeIban(transaction.counterpartyIban) : '';

    for (const member of members) {
        const first = normalizeText(member.firstName);
        const last = normalizeText(member.lastName);
        const email = normalizeCompact(member.email);
        const memberIban = member.iban ? normalizeIban(member.iban) : '';

        if (!first || !last || !member.email) {
            continue;
        }

        const fullName = `${first} ${last}`;
        let score = 0;

        if (txIban && memberIban && txIban === memberIban) {
            score += 1000;
        }

        if (email && compactText.includes(email)) {
            score += 600;
        }

        if (hasWord(normalizedText, String(member.number))) {
            score += 450;
        }

        const hasFullName = normalizedText.includes(fullName);
        const hasFirst = normalizedText.includes(first);
        const hasLast = normalizedText.includes(last);
        if (hasFullName) {
            score += 220;
        } else if (hasFirst && hasLast) {
            score += 140;
        }

        if (score === 0) {
            continue;
        }

        if (!bestMatch || score > bestMatch.score) {
            bestMatch = { member, score };
        }
    }

    return bestMatch?.member ?? null;
}

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

    router.post('/transactions/import/camt053', verifyToken, asyncHandler(async (req: AuthenticatedRequest, res) => {
        const { xml } = req.body as { xml?: string };
        const imported = parseCamt053(xml ?? '');
        const prisma = await getClient();
        const members = await prisma.member.findMany({
            select: { id: true, firstName: true, lastName: true, number: true, email: true, iban: true },
        });

        let createdCount = 0;
        let skippedCount = 0;
        let matchedMemberCount = 0;
        let memberFeesMarkedPaid = 0;
        let memberFeesCreated = 0;

        for (const tx of imported) {
            const duplicate = await prisma.financeTransaction.findFirst({
                where: {
                    date: tx.date,
                    amount: tx.amount,
                    type: tx.type,
                    description: tx.description,
                },
                select: { id: true },
            });

            if (duplicate) {
                skippedCount += 1;
                continue;
            }

            await prisma.financeTransaction.create({
                data: {
                    date: tx.date,
                    description: tx.description,
                    amount: tx.amount,
                    type: tx.type,
                    category: tx.category,
                    notes: tx.notes,
                },
            });
            createdCount += 1;

            if (tx.type !== 'income') {
                continue;
            }

            const matchedMember = findMatchingMember(tx, members);
            if (!matchedMember) {
                continue;
            }

            matchedMemberCount += 1;

            const txYear = tx.date.getFullYear();
            const existingOpenFee = await prisma.memberFee.findFirst({
                where: {
                    memberId: matchedMember.id,
                    year: txYear,
                    paidDate: null,
                    amount: {
                        gte: tx.amount - 0.005,
                        lte: tx.amount + 0.005,
                    },
                },
                orderBy: { dueDate: 'asc' },
            });

            if (existingOpenFee) {
                await prisma.memberFee.update({
                    where: { id: existingOpenFee.id },
                    data: {
                        paidDate: tx.date,
                        description: existingOpenFee.description ?? `Auto-matched CAMT.053: ${tx.description}`,
                    },
                });
                memberFeesMarkedPaid += 1;
                continue;
            }

            const alreadyPaidSameYear = await prisma.memberFee.findFirst({
                where: {
                    memberId: matchedMember.id,
                    year: txYear,
                    paidDate: {
                        not: null,
                    },
                    amount: {
                        gte: tx.amount - 0.005,
                        lte: tx.amount + 0.005,
                    },
                },
                select: { id: true },
            });

            if (alreadyPaidSameYear) {
                continue;
            }

            await prisma.memberFee.create({
                data: {
                    memberId: matchedMember.id,
                    amount: tx.amount,
                    dueDate: tx.date,
                    paidDate: tx.date,
                    description: `Imported from CAMT.053: ${tx.description}`,
                    year: txYear,
                },
            });
            memberFeesCreated += 1;
        }

        await createAuditLog(prisma, 'CREATE', 'FinanceTransactionImport', 0, req.userEmail ?? '', {
            format: 'CAMT.053',
            importedCount: createdCount,
            skippedCount,
            totalCount: imported.length,
            matchedMemberCount,
            memberFeesMarkedPaid,
            memberFeesCreated,
        });

        res.status(201).json({
            importedCount: createdCount,
            skippedCount,
            totalCount: imported.length,
            matchedMemberCount,
            memberFeesMarkedPaid,
            memberFeesCreated,
        });
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

    router.post('/reset', verifyToken, asyncHandler(async (req: AuthenticatedRequest, res) => {
        const { confirmation } = req.body as { confirmation?: string };
        if (confirmation !== 'DELETE_KASSENBUCH') {
            res.status(400).json({ error: 'Confirmation token is invalid.' });
            return;
        }

        const prisma = await getClient();

        const [deletedMemberFees, deletedTransactions] = await prisma.$transaction([
            prisma.memberFee.deleteMany(),
            prisma.financeTransaction.deleteMany(),
        ]);

        await createAuditLog(prisma, 'DELETE', 'FinanceLedger', 0, req.userEmail ?? '', {
            deletedMemberFees: deletedMemberFees.count,
            deletedTransactions: deletedTransactions.count,
        });

        res.json({
            deletedMemberFees: deletedMemberFees.count,
            deletedTransactions: deletedTransactions.count,
        });
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


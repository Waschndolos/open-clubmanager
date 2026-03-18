import api from './api';
import { FinanceTransaction, MemberFee } from './types';

// ─── Finance Transactions ─────────────────────────────────────────────────────

export async function fetchTransactions(): Promise<FinanceTransaction[]> {
    const res = await api.get<FinanceTransaction[]>('/finance/transactions');
    return res.data;
}

export async function createTransaction(
    data: Omit<FinanceTransaction, 'id' | 'createdAt' | 'updatedAt'>
): Promise<FinanceTransaction> {
    const res = await api.post<FinanceTransaction>('/finance/transactions', data);
    return res.data;
}

export async function updateTransaction(
    data: FinanceTransaction
): Promise<FinanceTransaction> {
    const res = await api.put<FinanceTransaction>(`/finance/transactions/${data.id}`, data);
    return res.data;
}

export async function deleteTransaction(id: number): Promise<void> {
    await api.delete(`/finance/transactions/${id}`);
}

// ─── Member Fees ──────────────────────────────────────────────────────────────

export async function fetchMemberFees(): Promise<MemberFee[]> {
    const res = await api.get<MemberFee[]>('/finance/memberfees');
    return res.data;
}

export async function createMemberFee(
    data: Omit<MemberFee, 'id' | 'member' | 'createdAt' | 'updatedAt'>
): Promise<MemberFee> {
    const res = await api.post<MemberFee>('/finance/memberfees', data);
    return res.data;
}

export async function updateMemberFee(data: MemberFee): Promise<MemberFee> {
    const res = await api.put<MemberFee>(`/finance/memberfees/${data.id}`, data);
    return res.data;
}

export async function deleteMemberFee(id: number): Promise<void> {
    await api.delete(`/finance/memberfees/${id}`);
}

import { FinanceTransaction, MemberFee } from './types';
import { getDataClient } from './clientFactory';

// ─── Finance Transactions ─────────────────────────────────────────────────────

export async function fetchTransactions(): Promise<FinanceTransaction[]> {
    return getDataClient().finance.listTransactions();
}

export async function createTransaction(
    data: Omit<FinanceTransaction, 'id' | 'createdAt' | 'updatedAt'>
): Promise<FinanceTransaction> {
    return getDataClient().finance.createTransaction(data);
}

export async function updateTransaction(
    data: FinanceTransaction
): Promise<FinanceTransaction> {
    return getDataClient().finance.updateTransaction(data);
}

export async function deleteTransaction(id: number): Promise<void> {
    return getDataClient().finance.deleteTransaction(id);
}

// ─── Member Fees ──────────────────────────────────────────────────────────────

export async function fetchMemberFees(): Promise<MemberFee[]> {
    return getDataClient().finance.listMemberFees();
}

export async function createMemberFee(
    data: Omit<MemberFee, 'id' | 'member' | 'createdAt' | 'updatedAt'>
): Promise<MemberFee> {
    return getDataClient().finance.createMemberFee(data);
}

export async function updateMemberFee(data: MemberFee): Promise<MemberFee> {
    return getDataClient().finance.updateMemberFee(data);
}

export async function deleteMemberFee(id: number): Promise<void> {
    return getDataClient().finance.deleteMemberFee(id);
}

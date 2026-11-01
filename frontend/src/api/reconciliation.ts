import api from './api';
import { HonorMember } from './honors';

/** How sure the matching is about a suggestion. */
export type Confidence = 'certain' | 'probable' | 'unclear';

/** Why a suggestion was made. */
export type MatchReason =
    | 'END_TO_END_ID'
    | 'MEMBER_NUMBER'
    | 'IBAN'
    | 'NAME'
    | 'AMOUNT'
    | 'HOUSEHOLD'
    | 'PARTIAL'
    | 'OVERPAID';

/**
 * open: waits for review; reconciled: fees assigned; dismissed: reviewed
 * without a fee; none: an expense, nothing to reconcile.
 */
export type ReconciliationStatus = 'open' | 'reconciled' | 'dismissed' | 'none';

export interface FeeAllocation {
    feeId: string;
    amount: number;
}

export interface ReconciliationSuggestion {
    confidence: Confidence;
    reasons: MatchReason[];
    allocations: FeeAllocation[];
    /** Part of the transaction not assigned to a fee (overpayment). */
    unallocated: number;
}

export interface FeePayment {
    id: string;
    feeId: string;
    member: HonorMember;
    year: number;
    description?: string;
    amount: number;
}

/** An imported bank transaction. */
export interface ReconciliationEntry {
    transactionId: string;
    date: string;
    description: string;
    amount: number;
    type: 'income' | 'expense';
    counterpartyName?: string;
    counterpartyIban?: string;
    endToEndId?: string;
    /** A returned direct debit (Rücklastschrift). */
    isReturn: boolean;
    returnReason?: string;
    status: ReconciliationStatus;
    suggestion?: ReconciliationSuggestion;
    payments: FeePayment[];
}

/** A fee with what has been paid on it. */
export interface ReconciliationFee {
    feeId: string;
    member: HonorMember;
    year: number;
    description?: string;
    dueDate: string;
    amount: number;
    paid: number;
    paidDate?: string;
    /** Can take a payment: unpaid, not in a pending direct debit run. */
    open: boolean;
    inCollection: boolean;
}

export interface Reconciliation {
    entries: ReconciliationEntry[];
    fees: ReconciliationFee[];
}

export async function fetchReconciliation(): Promise<Reconciliation> {
    return (await api.get<Reconciliation>('/reconciliation')).data;
}

export async function confirmReconciliation(transactionId: string, allocations: FeeAllocation[]): Promise<ReconciliationEntry> {
    return (await api.post<ReconciliationEntry>(`/reconciliation/${encodeURIComponent(transactionId)}/confirm`, { allocations })).data;
}

/** Confirms every suggestion rated certain; returns how many. */
export async function confirmCertainReconciliations(): Promise<number> {
    return (await api.post<{ confirmed: number }>('/reconciliation/confirm-certain')).data.confirmed;
}

export async function dismissReconciliation(transactionId: string): Promise<ReconciliationEntry> {
    return (await api.post<ReconciliationEntry>(`/reconciliation/${encodeURIComponent(transactionId)}/dismiss`)).data;
}

/** Undoes the assignment or dismissal of a transaction. */
export async function reopenReconciliation(transactionId: string): Promise<ReconciliationEntry> {
    return (await api.post<ReconciliationEntry>(`/reconciliation/${encodeURIComponent(transactionId)}/reopen`)).data;
}

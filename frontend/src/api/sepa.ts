import api from './api';
import { HonorMember } from './honors';

/** The club as creditor of its SEPA direct debits. */
export interface SepaCreditor {
    name: string;
    creditorId: string;
    iban: string;
    bic?: string;
    remittanceTemplate?: string;
}

export type SequenceType = 'FRST' | 'RCUR';

/** Problems that keep a fee out of a direct debit run. */
export type SepaIssue =
    | 'MISSING_IBAN'
    | 'INVALID_IBAN'
    | 'INVALID_BIC'
    | 'MISSING_MANDATE'
    | 'MISSING_MANDATE_REFERENCE'
    | 'INVALID_MANDATE_REFERENCE'
    | 'INVALID_AMOUNT';

/** An open fee (unpaid, not in a run) with the member's bank and mandate data. */
export interface SepaCandidate {
    feeId: string;
    member: HonorMember;
    amount: number;
    year: number;
    description?: string;
    dueDate: string;
    debtorName: string;
    iban?: string;
    mandateDate?: string;
    mandateReference?: string;
    sequenceType: SequenceType;
    /** Remittance text rendered from the template. */
    remittance: string;
    issues: SepaIssue[];
}

export interface SepaRunItem {
    feeId: string;
    member: HonorMember;
    amount: number;
    sequenceType: SequenceType;
    mandateReference: string;
    debtorName: string;
    iban: string;
    remittance: string;
}

export interface SepaRun {
    id: string;
    collectionDate: string;
    /** Set once the collection has been confirmed; the fees are then paid. */
    paidDate?: string;
    createdAt: string;
    total: number;
    count: number;
    items: SepaRunItem[];
}

export const REMITTANCE_PLACEHOLDERS = ['{description}', '{year}', '{number}', '{firstName}', '{lastName}', '{amount}'];

export async function fetchSepaCreditor(): Promise<SepaCreditor | null> {
    const res = await api.get<SepaCreditor | ''>('/sepa/creditor');
    return res.status === 204 || !res.data ? null : res.data;
}

export async function saveSepaCreditor(data: SepaCreditor): Promise<SepaCreditor> {
    return (await api.put<SepaCreditor>('/sepa/creditor', data)).data;
}

export async function fetchSepaMandate(memberId: string): Promise<string | undefined> {
    const res = await api.get<{ mandateReference?: string | null }>(`/members/${encodeURIComponent(memberId)}/sepa-mandate`);
    return res.data.mandateReference ?? undefined;
}

export async function saveSepaMandate(memberId: string, mandateReference: string): Promise<void> {
    await api.put(`/members/${encodeURIComponent(memberId)}/sepa-mandate`, { mandateReference: mandateReference || null });
}

/** Lists open fees; template overrides the remittance template of the settings. */
export async function fetchSepaCandidates(template?: string): Promise<SepaCandidate[]> {
    const params = template === undefined ? {} : { template };
    return (await api.get<SepaCandidate[]>('/sepa/candidates', { params })).data;
}

export async function fetchSepaRuns(): Promise<SepaRun[]> {
    return (await api.get<SepaRun[]>('/sepa/runs')).data;
}

export async function createSepaRun(data: { collectionDate: string; remittanceTemplate?: string; feeIds: string[] }): Promise<SepaRun> {
    return (await api.post<SepaRun>('/sepa/runs', data)).data;
}

export async function confirmSepaRun(id: string, paidDate: string): Promise<SepaRun> {
    return (await api.post<SepaRun>(`/sepa/runs/${encodeURIComponent(id)}/confirm`, { paidDate })).data;
}

export async function cancelSepaRun(id: string): Promise<void> {
    await api.delete(`/sepa/runs/${encodeURIComponent(id)}`);
}

/** Downloads the pain.008 file of a run for upload to the bank. */
export async function downloadSepaRun(run: SepaRun): Promise<void> {
    const res = await api.get(`/sepa/runs/${encodeURIComponent(run.id)}/xml`, { responseType: 'blob' });
    const href = URL.createObjectURL(new Blob([res.data], { type: 'application/xml' }));
    const anchor = document.createElement('a');
    anchor.href = href;
    anchor.download = `lastschrift-${run.collectionDate}-${run.id.slice(-8)}.xml`;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(href);
}

/** Checks the IBAN check digits (ISO 13616, mod 97), for form feedback. */
export function isValidIban(value: string): boolean {
    const iban = value.replace(/\s+/g, '').toUpperCase();
    if (!/^[A-Z]{2}[0-9]{2}[A-Z0-9]{11,30}$/.test(iban)) return false;
    const digits = (iban.slice(4) + iban.slice(0, 4)).replace(/[A-Z]/g, (c) => String(c.charCodeAt(0) - 55));
    let rest = 0;
    for (const d of digits) rest = (rest * 10 + Number(d)) % 97;
    return rest === 1;
}

import api from './api';
import { HonorMember } from './honors';

/**
 * notice: Freistellungsbescheid or annex to the corporate tax assessment
 * (receipts recognised for 5 years); assessment: separate assessment of the
 * statutes under § 60a AO (3 years).
 */
export type ExemptionKind = 'notice' | 'assessment';

/** The club's data printed on donation receipts. */
export interface DonationSettings {
    issuerName: string;
    issuerAddress: string;
    taxOffice: string;
    taxNumber: string;
    exemptionKind: ExemptionKind;
    exemptionDate: string;
    assessmentPeriod?: string;
    purposes: string;
    /** Membership fees may be certified (not for sports clubs). */
    feesDeductible: boolean;
    signatory?: string;
    place?: string;
    /** Read only: receipts issued today would no longer be recognised. */
    exemptionExpired?: boolean;
    /** Read only: last day receipts based on the exemption are recognised. */
    validUntil?: string;
}

/** A donor who is not a member. */
export interface Donor {
    id: string;
    name: string;
    street?: string;
    postalCode?: string;
    city?: string;
    notes?: string;
}

export interface DonorRef {
    id: string;
    name: string;
}

/** An income that is, or looks like, a donation. */
export interface Donation {
    transactionId: string;
    date: string;
    description: string;
    amount: number;
    category?: string;
    member?: HonorMember;
    donor?: DonorRef;
    /** Waiver of reimbursement of expenses (Aufwandsspende). */
    expenseWaiver: boolean;
    receiptNumber?: string;
}

export type ReceiptItemKind = 'donation' | 'membership_fee';

export interface CandidateItem {
    /** Transaction id of a donation or fee id of a paid fee. */
    ref: string;
    kind: ReceiptItemKind;
    date: string;
    amount: number;
    expenseWaiver: boolean;
    description: string;
    receiptId?: string;
    receiptNumber?: string;
}

/** A member or donor with the donations (and paid fees) of a year. */
export interface DonationCandidate {
    key: string;
    member?: HonorMember;
    donor?: DonorRef;
    name: string;
    /** Postal address; empty when street or town is missing. */
    address: string;
    items: CandidateItem[];
}

export interface ReceiptItem {
    date: string;
    kind: ReceiptItemKind;
    amount: number;
    expenseWaiver: boolean;
    transactionId?: string;
    feeId?: string;
}

/** An issued receipt; the list of receipts is the log. */
export interface DonationReceipt {
    id: string;
    number: string;
    year: number;
    kind: 'single' | 'collective';
    memberId?: string;
    donorId?: string;
    recipientName: string;
    recipientAddress: string;
    periodStart: string;
    periodEnd: string;
    total: number;
    issueDate: string;
    /** Set on a duplicate: the id of the original receipt. */
    duplicateOf?: string;
    createdBy?: string;
    createdAt: string;
    voidedAt?: string;
    voidReason?: string;
    items: ReceiptItem[];
}

export interface AnnualRunResult {
    created: DonationReceipt[];
    /** Donors left out because their address is incomplete. */
    skipped: DonationCandidate[];
}

export async function fetchDonationSettings(): Promise<DonationSettings | null> {
    const res = await api.get<DonationSettings | ''>('/donations/settings');
    return res.status === 204 || !res.data ? null : res.data;
}

export async function saveDonationSettings(data: DonationSettings): Promise<DonationSettings> {
    return (await api.put<DonationSettings>('/donations/settings', data)).data;
}

export async function fetchDonors(): Promise<Donor[]> {
    return (await api.get<Donor[]>('/donations/donors')).data;
}

export async function createDonor(data: Omit<Donor, 'id'>): Promise<Donor> {
    return (await api.post<Donor>('/donations/donors', data)).data;
}

export async function updateDonor(id: string, data: Omit<Donor, 'id'>): Promise<Donor> {
    return (await api.put<Donor>(`/donations/donors/${encodeURIComponent(id)}`, data)).data;
}

export async function deleteDonor(id: string): Promise<void> {
    await api.delete(`/donations/donors/${encodeURIComponent(id)}`);
}

export async function fetchDonations(): Promise<Donation[]> {
    return (await api.get<Donation[]>('/donations')).data;
}

/** Marks an income as a donation of a member or an external donor. */
export async function assignDonation(transactionId: string, data: { memberId?: string; donorId?: string; expenseWaiver: boolean }): Promise<void> {
    await api.put(`/donations/${encodeURIComponent(transactionId)}`, data);
}

export async function unassignDonation(transactionId: string): Promise<void> {
    await api.delete(`/donations/${encodeURIComponent(transactionId)}`);
}

export async function fetchDonationCandidates(year: number): Promise<DonationCandidate[]> {
    return (await api.get<DonationCandidate[]>('/donations/candidates', { params: { year } })).data;
}

export async function fetchDonationReceipts(year?: number): Promise<DonationReceipt[]> {
    return (await api.get<DonationReceipt[]>('/donations/receipts', { params: year ? { year } : {} })).data;
}

/**
 * Issues a receipt: single for one item, collective for several. Items that
 * are exactly those of an issued receipt yield a duplicate of it.
 */
export async function createDonationReceipt(data: { memberId?: string; donorId?: string; items: string[]; issueDate?: string }): Promise<DonationReceipt> {
    return (await api.post<DonationReceipt>('/donations/receipts', data)).data;
}

export async function createAnnualReceipts(year: number, issueDate?: string): Promise<AnnualRunResult> {
    return (await api.post<AnnualRunResult>('/donations/receipts/annual', { year, issueDate })).data;
}

export async function duplicateDonationReceipt(id: string): Promise<DonationReceipt> {
    return (await api.post<DonationReceipt>(`/donations/receipts/${encodeURIComponent(id)}/duplicate`)).data;
}

export async function voidDonationReceipt(id: string, reason: string): Promise<DonationReceipt> {
    return (await api.post<DonationReceipt>(`/donations/receipts/${encodeURIComponent(id)}/void`, { reason })).data;
}

/** Downloads one or more receipts as a single PDF. */
export async function downloadDonationReceipts(receipts: DonationReceipt[]): Promise<void> {
    if (receipts.length === 0) return;
    const ids = receipts.map((r) => r.id).join(',');
    const res = await api.get('/donations/receipts/pdf', { params: { ids }, responseType: 'blob' });
    const href = URL.createObjectURL(new Blob([res.data], { type: 'application/pdf' }));
    const single = receipts.length === 1 ? receipts[0] : undefined;
    const name = single ? `${single.number}${single.duplicateOf ? '-duplikat' : ''}` : new Date().toISOString().slice(0, 10);
    const anchor = document.createElement('a');
    anchor.href = href;
    anchor.download = `zuwendungsbestaetigung-${name}.pdf`;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(href);
}

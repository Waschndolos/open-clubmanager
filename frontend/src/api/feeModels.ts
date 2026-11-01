import api from './api';
import { HonorMember } from './honors';

export type FeeInterval = 'yearly' | 'half_yearly' | 'quarterly' | 'monthly';

export const FEE_INTERVALS: FeeInterval[] = ['yearly', 'half_yearly', 'quarterly', 'monthly'];

/** A fee class (adults, youth, family, …) with its amount per period. */
export interface FeeModel {
    id: string;
    name: string;
    amount: number;
    interval: FeeInterval;
    /** Inclusive age limits; models with one are picked automatically by age. */
    minAge?: number;
    maxAge?: number;
    /** Charge only the months in which the member belongs to the club. */
    proRata: boolean;
}

/** auto: by age; model: a fixed model; exempt: no fee. */
export type FeeMode = 'auto' | 'model' | 'exempt';

export interface FeeAssignment {
    mode: FeeMode;
    feeModelId?: string;
}

/** Value of an assignment select: "auto", "exempt" or a model id. */
export const assignmentValue = (a: { mode: FeeMode; model?: FeeModel; feeModelId?: string }): string =>
    a.mode === 'model' ? (a.feeModelId ?? a.model?.id ?? 'auto') : a.mode;

export const toAssignment = (value: string): FeeAssignment =>
    value === 'auto' || value === 'exempt' ? { mode: value } : { mode: 'model', feeModelId: value };

export interface MemberFeeModel {
    member: HonorMember;
    mode: FeeMode;
    model?: FeeModel;
    /** Age on 1 January, which decides the model for the year. */
    age?: number;
}

export interface FeePeriod {
    start: string;
    end: string;
    /** "2027", "2027-H1", "2027-Q3" or "2027-05". */
    label: string;
}

export interface FeeRunItem {
    member: HonorMember;
    model: FeeModel;
    period: FeePeriod;
    amount: number;
    proRata: boolean;
    dueDate: string;
    description: string;
}

export type FeeSkipReason = 'NO_MODEL' | 'EXEMPT' | 'ALREADY_CHARGED';

export interface FeeRunSkip {
    member: HonorMember;
    reason: FeeSkipReason;
    model?: FeeModel;
    period?: FeePeriod;
}

export interface FeeRunPlan {
    count: number;
    total: number;
    items: FeeRunItem[];
    skipped: FeeRunSkip[];
}

export interface FeeRun {
    id: string;
    /** First and last month (YYYY-MM) in which the charged periods start. */
    from: string;
    to: string;
    dueDate: string;
    createdAt: string;
    count: number;
    total: number;
    paidCount: number;
}

export interface FeeRunInput {
    from: string;
    to: string;
    dueDate?: string;
}

export async function fetchFeeModels(): Promise<FeeModel[]> {
    return (await api.get<FeeModel[]>('/fee-models')).data;
}

export async function createFeeModel(data: Omit<FeeModel, 'id'>): Promise<FeeModel> {
    return (await api.post<FeeModel>('/fee-models', data)).data;
}

export async function updateFeeModel(data: FeeModel): Promise<FeeModel> {
    return (await api.put<FeeModel>(`/fee-models/${encodeURIComponent(data.id)}`, data)).data;
}

export async function deleteFeeModel(id: string): Promise<void> {
    await api.delete(`/fee-models/${encodeURIComponent(id)}`);
}

/** The fee model of every member on a day (YYYY-MM-DD, default today). */
export async function fetchMemberFeeModels(date?: string): Promise<MemberFeeModel[]> {
    return (await api.get<MemberFeeModel[]>('/fee-models/members', { params: date ? { date } : {} })).data;
}

export async function fetchFeeAssignment(memberId: string): Promise<FeeAssignment> {
    return (await api.get<FeeAssignment>(`/members/${encodeURIComponent(memberId)}/fee-model`)).data;
}

export async function saveFeeAssignment(memberId: string, data: FeeAssignment): Promise<FeeAssignment> {
    return (await api.put<FeeAssignment>(`/members/${encodeURIComponent(memberId)}/fee-model`, data)).data;
}

export async function previewFeeRun(data: FeeRunInput): Promise<FeeRunPlan> {
    return (await api.post<FeeRunPlan>('/fee-runs/preview', data)).data;
}

export async function createFeeRun(data: FeeRunInput): Promise<FeeRun> {
    return (await api.post<FeeRun>('/fee-runs', data)).data;
}

export async function fetchFeeRuns(): Promise<FeeRun[]> {
    return (await api.get<FeeRun[]>('/fee-runs')).data;
}

/** Removes the open fees of a run; returns how many were removed. */
export async function deleteFeeRun(id: string): Promise<number> {
    return (await api.delete<{ removed: number }>(`/fee-runs/${encodeURIComponent(id)}`)).data.removed;
}

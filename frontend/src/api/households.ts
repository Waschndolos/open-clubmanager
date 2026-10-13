import api from './api';
import { HonorMember } from './honors';

export interface HouseholdMember extends HonorMember {
    birthday?: string;
    exitDate?: string;
    /** False once the member has left the club. */
    active: boolean;
}

export interface Household {
    id: string;
    name: string;
    payer?: HonorMember;
    shareAddress: boolean;
    shareBank: boolean;
    members: HouseholdMember[];
    /** Active members remain without an active main payer (e.g. the payer left). */
    payerMissing: boolean;
}

export interface HouseholdInput {
    name: string;
    payerId?: string;
    shareAddress: boolean;
    shareBank: boolean;
    memberIds: string[];
}

export interface Guardian {
    id: string;
    /** Set when the guardian is a member of the club. */
    member?: HonorMember;
    name?: string;
    relation?: string;
    email?: string;
    phone?: string;
}

export interface GuardianInput {
    memberId?: string;
    name?: string;
    relation?: string;
    email?: string;
    phone?: string;
}

export async function fetchHouseholds(): Promise<Household[]> {
    return (await api.get<Household[]>('/households')).data;
}

export async function createHousehold(data: HouseholdInput): Promise<Household> {
    return (await api.post<Household>('/households', data)).data;
}

export async function updateHousehold(id: string, data: HouseholdInput): Promise<Household> {
    return (await api.put<Household>(`/households/${encodeURIComponent(id)}`, data)).data;
}

export async function deleteHousehold(id: string): Promise<void> {
    await api.delete(`/households/${encodeURIComponent(id)}`);
}

export async function fetchGuardians(memberId: string): Promise<Guardian[]> {
    return (await api.get<Guardian[]>(`/members/${encodeURIComponent(memberId)}/guardians`)).data;
}

export async function createGuardian(memberId: string, data: GuardianInput): Promise<void> {
    await api.post(`/members/${encodeURIComponent(memberId)}/guardians`, data);
}

export async function deleteGuardian(id: string): Promise<void> {
    await api.delete(`/guardians/${encodeURIComponent(id)}`);
}

/** Whether a member with the given birthday (ISO date) is under 18 today. */
export function isMinor(birthday?: string | Date): boolean {
    if (!birthday) return false;
    const b = new Date(birthday);
    if (Number.isNaN(b.getTime())) return false;
    const adult = new Date(b.getFullYear() + 18, b.getMonth(), b.getDate());
    return adult > new Date();
}

export function memberName(m: HonorMember): string {
    return `${m.firstName} ${m.lastName}`;
}

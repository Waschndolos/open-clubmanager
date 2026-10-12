import api from './api';

export type HonorKind = 'membership' | 'birthday' | 'other';

export interface HonorMember {
    id: string;
    number: number;
    firstName: string;
    lastName: string;
}

export interface Jubilee {
    member: HonorMember;
    kind: 'membership' | 'birthday';
    /** Membership years or age reached. */
    milestone: number;
    /** YYYY-MM-DD */
    date: string;
}

export interface Honor {
    id: string;
    member: HonorMember;
    kind: HonorKind;
    milestone?: number;
    title: string;
    date: string;
    note?: string;
}

export interface JubileeQuery {
    from: string;
    to: string;
    years: number[];
    ages: number[];
}

function params(q: JubileeQuery) {
    return { from: q.from, to: q.to, years: q.years.join(','), ages: q.ages.join(',') };
}

/** Anniversaries and round birthdays in the range that have not been honored yet. */
export async function fetchJubilees(q: JubileeQuery): Promise<Jubilee[]> {
    const res = await api.get<Jubilee[]>('/jubilees', { params: params(q) });
    return res.data;
}

/** Downloads the jubilee list as CSV (for the general meeting or certificates). */
export async function downloadJubileesCsv(q: JubileeQuery): Promise<void> {
    const res = await api.get('/jubilees', { params: { ...params(q), format: 'csv' }, responseType: 'blob' });
    const url = URL.createObjectURL(new Blob([res.data], { type: 'text/csv' }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `jubilees-${q.from}-${q.to}.csv`;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(url);
}

export async function fetchHonors(): Promise<Honor[]> {
    const res = await api.get<Honor[]>('/honors');
    return res.data;
}

export async function createHonor(
    memberId: string,
    data: { kind: HonorKind; milestone?: number; title: string; date: string; note?: string }
): Promise<Honor> {
    const res = await api.post<Honor>(`/members/${encodeURIComponent(memberId)}/honors`, data);
    return res.data;
}

export async function deleteHonor(id: string): Promise<void> {
    await api.delete(`/honors/${encodeURIComponent(id)}`);
}

export async function fetchMembershipAdjust(memberId: string): Promise<number> {
    const res = await api.get<{ years: number }>(`/members/${encodeURIComponent(memberId)}/membership-adjustment`);
    return res.data.years;
}

export async function setMembershipAdjust(memberId: string, years: number): Promise<void> {
    await api.put(`/members/${encodeURIComponent(memberId)}/membership-adjustment`, { years });
}

export const DEFAULT_YEARS = [10, 25, 40, 50, 60, 70];
export const DEFAULT_AGES = [50, 60, 70, 75, 80, 85, 90, 95, 100];

/** Formats a local date as YYYY-MM-DD. */
export function isoDate(d: Date): string {
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

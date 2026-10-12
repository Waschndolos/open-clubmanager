import api from './api';
import { HonorMember } from './honors';

export type QualificationStatus = 'valid' | 'expiring' | 'expired';

export interface QualificationType {
    id: string;
    name: string;
    validityMonths?: number;
    /** Only the inspection date is stored (police clearance certificate). */
    inspectionOnly: boolean;
}

export interface Qualification {
    id: string;
    member: HonorMember;
    type: QualificationType;
    issuedDate: string;
    expiryDate?: string;
    note?: string;
    status: QualificationStatus;
}

export async function fetchQualificationTypes(): Promise<QualificationType[]> {
    return (await api.get<QualificationType[]>('/qualification-types')).data;
}

export async function saveQualificationType(type: Partial<QualificationType> & { name: string }): Promise<QualificationType> {
    const body = { name: type.name, validityMonths: type.validityMonths, inspectionOnly: !!type.inspectionOnly };
    if (type.id) return (await api.put<QualificationType>(`/qualification-types/${encodeURIComponent(type.id)}`, body)).data;
    return (await api.post<QualificationType>('/qualification-types', body)).data;
}

export async function deleteQualificationType(id: string): Promise<void> {
    await api.delete(`/qualification-types/${encodeURIComponent(id)}`);
}

export async function fetchQualifications(warnDays = 60): Promise<Qualification[]> {
    return (await api.get<Qualification[]>('/qualifications', { params: { warnDays } })).data;
}

export async function createQualification(
    memberId: string,
    data: { typeId: string; issuedDate: string; expiryDate?: string; note?: string }
): Promise<void> {
    await api.post(`/members/${encodeURIComponent(memberId)}/qualifications`, data);
}

export async function deleteQualification(id: string): Promise<void> {
    await api.delete(`/qualifications/${encodeURIComponent(id)}`);
}

import { ClubSection, MemberSummary } from './types';
import { createEntityApi } from './entityApi';
import api from './api';

const entityApi = createEntityApi<ClubSection>('sections');

export const fetchSections = entityApi.fetchAll;
export const createSection = entityApi.create;
export const updateSection = entityApi.update;
export const deleteSection = entityApi.delete;

export async function fetchSectionMembers(id: string): Promise<MemberSummary[]> {
    const res = await api.get<MemberSummary[]>(`/sections/${id}/members`);
    return res.data;
}

export async function assignSectionMembers(id: string, memberIds: string[]): Promise<void> {
    await api.put(`/sections/${id}/members`, { memberIds });
}
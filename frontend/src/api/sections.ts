import { ClubSection, MemberSummary } from './types';
import { createEntityApi } from './entityApi';
import api, { BACKEND_URL } from './api';

const entityApi = createEntityApi<ClubSection>('sections');

export const fetchSections = entityApi.fetchAll;
export const createSection = entityApi.create;
export const updateSection = entityApi.update;
export const deleteSection = entityApi.delete;

export async function fetchSectionMembers(id: number): Promise<MemberSummary[]> {
    const res = await fetch(`${BACKEND_URL}/sections/${id}/members`);
    if (!res.ok) throw new Error(`Error fetching members for section ${id}.`);
    return res.json();
}

export async function assignSectionMembers(id: number, memberIds: number[]): Promise<void> {
    await api.put(`/sections/${id}/members`, { memberIds });
}
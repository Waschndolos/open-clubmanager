import { Group, MemberSummary } from './types';
import { createEntityApi } from './entityApi';
import api, { BACKEND_URL } from './api';

const entityApi = createEntityApi<Group>('groups');

export const fetchGroups = entityApi.fetchAll;
export const createGroup = entityApi.create;
export const updateGroup = entityApi.update;
export const deleteGroup = entityApi.delete;

export async function fetchGroupMembers(id: number): Promise<MemberSummary[]> {
    const res = await fetch(`${BACKEND_URL}/groups/${id}/members`);
    if (!res.ok) throw new Error(`Error fetching members for group ${id}.`);
    return res.json();
}

export async function assignGroupMembers(id: number, memberIds: number[]): Promise<void> {
    await api.put(`/groups/${id}/members`, { memberIds });
}

import { Group, MemberSummary } from './types';
import { createEntityApi } from './entityApi';
import api from './api';

const entityApi = createEntityApi<Group>('groups');

export const fetchGroups = entityApi.fetchAll;
export const createGroup = entityApi.create;
export const updateGroup = entityApi.update;
export const deleteGroup = entityApi.delete;

export async function fetchGroupMembers(id: string): Promise<MemberSummary[]> {
    const res = await api.get<MemberSummary[]>(`/groups/${id}/members`);
    return res.data;
}

export async function assignGroupMembers(id: string, memberIds: string[]): Promise<void> {
    await api.put(`/groups/${id}/members`, { memberIds });
}

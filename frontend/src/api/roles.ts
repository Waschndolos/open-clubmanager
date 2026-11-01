import { Role, MemberSummary } from './types';
import { createEntityApi } from './entityApi';
import api from './api';

const entityApi = createEntityApi<Role>('roles');

export const fetchRoles = entityApi.fetchAll;
export const createRole = entityApi.create;
export const updateRole = entityApi.update;
export const deleteRole = entityApi.delete;

export async function fetchRoleMembers(id: string): Promise<MemberSummary[]> {
    const res = await api.get<MemberSummary[]>(`/roles/${id}/members`);
    return res.data;
}

export async function assignRoleMembers(id: string, memberIds: string[]): Promise<void> {
    await api.put(`/roles/${id}/members`, { memberIds });
}
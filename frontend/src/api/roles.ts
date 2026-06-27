import { Role, MemberSummary } from './types';
import { createEntityApi } from './entityApi';
import api, { BACKEND_URL } from './api';

const entityApi = createEntityApi<Role>('roles');

export const fetchRoles = entityApi.fetchAll;
export const createRole = entityApi.create;
export const updateRole = entityApi.update;
export const deleteRole = entityApi.delete;

export async function fetchRoleMembers(id: number): Promise<MemberSummary[]> {
    const res = await fetch(`${BACKEND_URL}/roles/${id}/members`);
    if (!res.ok) throw new Error(`Error fetching members for role ${id}.`);
    return res.json();
}

export async function assignRoleMembers(id: number, memberIds: number[]): Promise<void> {
    await api.put(`/roles/${id}/members`, { memberIds });
}
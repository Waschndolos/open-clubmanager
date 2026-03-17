import { Role } from './types';
import { createEntityApi } from './entityApi';

const api = createEntityApi<Role>('roles');

export const fetchRoles = api.fetchAll;
export const createRole = api.create;
export const updateRole = api.update;
export const deleteRole = api.delete;
import { Group } from './types';
import { createEntityApi } from './entityApi';

const api = createEntityApi<Group>('groups');

export const fetchGroups = api.fetchAll;
export const createGroup = api.create;
export const updateGroup = api.update;
export const deleteGroup = api.delete;

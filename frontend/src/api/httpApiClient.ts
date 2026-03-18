import api, { BACKEND_URL } from './api';
import { Member, Role, Group, ClubSection } from './types';
import { DataClient } from './dataClient';
import { LockInfo } from './ipcTypes';

function notSupported(method: string): never {
    throw new Error(`${method} is not supported in browser (HTTP) mode.`);
}

export const httpApiClient: DataClient = {
    club: {
        selectFolder: () => notSupported('club.selectFolder'),
        getFolder: async () => null,
        initFolder: async () => { /* no-op in HTTP mode */ },
    },

    members: {
        list: async (): Promise<Member[]> => {
            const res = await fetch(`${BACKEND_URL}/members`);
            if (!res.ok) throw new Error('Error fetching members.');
            return res.json();
        },
        get: async (id: number): Promise<Member> => {
            const res = await fetch(`${BACKEND_URL}/members/${id}`);
            if (!res.ok) throw new Error(`Error fetching member ${id}.`);
            return res.json();
        },
        create: async (data: Omit<Member, 'id'>): Promise<Member> => {
            const res = await api.post<Member>('/members', data);
            return res.data;
        },
        update: async (data: Member): Promise<Member> => {
            const res = await api.put<Member>(`/members/${data.id}`, data);
            return res.data;
        },
        delete: async (members: Member[]): Promise<void> => {
            for (const member of members) {
                await api.delete(`/members/${member.id}`);
            }
        },
        lock: async (_id: number, _owner: string): Promise<{ acquired: boolean; lock: LockInfo | null }> => {
            return { acquired: true, lock: null };
        },
        unlock: async (_id: number, _owner: string): Promise<{ released: boolean }> => {
            return { released: true };
        },
        getLock: async (_id: number): Promise<LockInfo | null> => null,
    },

    roles: {
        list: async (): Promise<Role[]> => {
            const res = await fetch(`${BACKEND_URL}/roles`);
            if (!res.ok) throw new Error('Error fetching roles.');
            return res.json();
        },
        create: async (data: Omit<Role, 'id'>): Promise<Role> => {
            const res = await api.post<Role>('/roles', data);
            return res.data;
        },
        update: async (data: Role): Promise<Role> => {
            const res = await api.put<Role>(`/roles/${data.id}`, data);
            return res.data;
        },
        delete: async (data: Role): Promise<void> => {
            await api.delete(`/roles/${data.id}`);
        },
    },

    groups: {
        list: async (): Promise<Group[]> => {
            const res = await fetch(`${BACKEND_URL}/groups`);
            if (!res.ok) throw new Error('Error fetching groups.');
            return res.json();
        },
        create: async (data: Omit<Group, 'id'>): Promise<Group> => {
            const res = await api.post<Group>('/groups', data);
            return res.data;
        },
        update: async (data: Group): Promise<Group> => {
            const res = await api.put<Group>(`/groups/${data.id}`, data);
            return res.data;
        },
        delete: async (data: Group): Promise<void> => {
            await api.delete(`/groups/${data.id}`);
        },
    },

    sections: {
        list: async (): Promise<ClubSection[]> => {
            const res = await fetch(`${BACKEND_URL}/sections`);
            if (!res.ok) throw new Error('Error fetching sections.');
            return res.json();
        },
        create: async (data: Omit<ClubSection, 'id'>): Promise<ClubSection> => {
            const res = await api.post<ClubSection>('/sections', data);
            return res.data;
        },
        update: async (data: ClubSection): Promise<ClubSection> => {
            const res = await api.put<ClubSection>(`/sections/${data.id}`, data);
            return res.data;
        },
        delete: async (data: ClubSection): Promise<void> => {
            await api.delete(`/sections/${data.id}`);
        },
    },
};

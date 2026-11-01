import api from './api';
import { Member, Role, Group, ClubSection } from './types';
import { DataClient } from './dataClient';

export const httpApiClient: DataClient = {
    members: {
        list: async (): Promise<Member[]> => {
            const res = await api.get<{ items?: Member[] } | Member[]>('/members', { params: { page: 1, pageSize: 500 } });
            const payload = res.data;
            return Array.isArray(payload) ? payload : (payload.items ?? []);
        },
        get: async (id: string): Promise<Member> => {
            const res = await api.get<Member>(`/members/${id}`);
            return res.data;
        },
        create: async (data: Omit<Member, 'id'>): Promise<Member> => {
            const res = await api.post<Member>('/members', data);
            return res.data;
        },
        update: async (data: Member): Promise<Member> => {
            const { versionToken, ...memberData } = data;
            const res = await api.put<Member>(`/members/${data.id}`, {
                ...memberData,
                expectedVersionToken: versionToken,
            });
            return res.data;
        },
        delete: async (members: Member[]): Promise<void> => {
            for (const member of members) {
                await api.delete(`/members/${member.id}`);
            }
        },
    },

    roles: {
        list: async (): Promise<Role[]> => {
            const res = await api.get<Role[]>('/roles');
            return res.data;
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
            const res = await api.get<Group[]>('/groups');
            return res.data;
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
            const res = await api.get<ClubSection[]>('/sections');
            return res.data;
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

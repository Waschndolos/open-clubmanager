import api, { BACKEND_URL } from './api';
import { Member, Role, Group, ClubSection, Event, EventAttendee, EventAttendanceStatus, EventType } from './types';
import { DataClient } from './dataClient';

export const httpApiClient: DataClient = {
    members: {
        list: async (): Promise<Member[]> => {
            const res = await fetch(`${BACKEND_URL}/members?page=1&pageSize=500`);
            if (!res.ok) throw new Error('Error fetching members.');
            const payload = await res.json() as { items?: Member[] } | Member[];
            return Array.isArray(payload) ? payload : (payload.items ?? []);
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

    events: {
        list: async (filters?: { startDateFrom?: string; startDateTo?: string; type?: EventType }): Promise<Event[]> => {
            const params = new URLSearchParams();
            if (filters?.startDateFrom) params.set('startDateFrom', filters.startDateFrom);
            if (filters?.startDateTo) params.set('startDateTo', filters.startDateTo);
            if (filters?.type) params.set('type', filters.type);
            const query = params.toString();
            const res = await api.get<Event[]>(`/events${query ? `?${query}` : ''}`);
            return res.data;
        },
        get: async (id: number): Promise<Event> => {
            const res = await api.get<Event>(`/events/${id}`);
            return res.data;
        },
        create: async (data: Omit<Event, 'id' | 'createdAt' | 'updatedAt'>): Promise<Event> => {
            const res = await api.post<Event>('/events', data);
            return res.data;
        },
        update: async (id: number, data: Partial<Omit<Event, 'id' | 'createdAt' | 'updatedAt'>>): Promise<Event> => {
            const res = await api.put<Event>(`/events/${id}`, data);
            return res.data;
        },
        delete: async (id: number): Promise<void> => {
            await api.delete(`/events/${id}`);
        },
        listAttendees: async (eventId: number): Promise<EventAttendee[]> => {
            const res = await api.get<EventAttendee[]>(`/events/${eventId}/attendees`);
            return res.data;
        },
        upsertAttendees: async (eventId: number, attendees: Array<{ memberId: number; status?: EventAttendanceStatus }>): Promise<void> => {
            await api.post(`/events/${eventId}/attendees`, { attendees });
        },
        removeAttendee: async (eventId: number, memberId: number): Promise<void> => {
            await api.delete(`/events/${eventId}/attendees/${memberId}`);
        },
    },
};

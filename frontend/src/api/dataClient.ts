import { Member, Role, Group, ClubSection, Event, EventAttendee, EventAttendanceStatus, EventType } from './types';

export interface MembersClient {
    list(): Promise<Member[]>;
    get(id: number): Promise<Member>;
    create(data: Omit<Member, 'id'>): Promise<Member>;
    update(data: Member): Promise<Member>;
    delete(members: Member[]): Promise<void>;
}

export interface RolesClient {
    list(): Promise<Role[]>;
    create(data: Omit<Role, 'id'>): Promise<Role>;
    update(data: Role): Promise<Role>;
    delete(data: Role): Promise<void>;
}

export interface GroupsClient {
    list(): Promise<Group[]>;
    create(data: Omit<Group, 'id'>): Promise<Group>;
    update(data: Group): Promise<Group>;
    delete(data: Group): Promise<void>;
}

export interface SectionsClient {
    list(): Promise<ClubSection[]>;
    create(data: Omit<ClubSection, 'id'>): Promise<ClubSection>;
    update(data: ClubSection): Promise<ClubSection>;
    delete(data: ClubSection): Promise<void>;
}

export interface EventsClient {
    list(filters?: { startDateFrom?: string; startDateTo?: string; type?: EventType }): Promise<Event[]>;
    get(id: number): Promise<Event>;
    create(data: Omit<Event, 'id' | 'createdAt' | 'updatedAt'>): Promise<Event>;
    update(id: number, data: Partial<Omit<Event, 'id' | 'createdAt' | 'updatedAt'>>): Promise<Event>;
    delete(id: number): Promise<void>;
    listAttendees(eventId: number): Promise<EventAttendee[]>;
    upsertAttendees(eventId: number, attendees: Array<{ memberId: number; status?: EventAttendanceStatus }>): Promise<void>;
    removeAttendee(eventId: number, memberId: number): Promise<void>;
}

export interface DataClient {
    members: MembersClient;
    roles: RolesClient;
    groups: GroupsClient;
    sections: SectionsClient;
    events: EventsClient;
}

import { Member, Role, Group, ClubSection } from './types';
import { LockInfo } from './ipcTypes';

export interface MembersClient {
    list(): Promise<Member[]>;
    get(id: number): Promise<Member>;
    create(data: Omit<Member, 'id'>): Promise<Member>;
    update(data: Member): Promise<Member>;
    delete(members: Member[]): Promise<void>;
    lock(id: number, owner: string): Promise<{ acquired: boolean; lock: LockInfo | null }>;
    unlock(id: number, owner: string): Promise<{ released: boolean }>;
    getLock(id: number): Promise<LockInfo | null>;
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

export interface ClubClient {
    selectFolder(): Promise<string | null>;
    getFolder(): Promise<string | null>;
    initFolder(): Promise<void>;
}

export interface DataClient {
    club: ClubClient;
    members: MembersClient;
    roles: RolesClient;
    groups: GroupsClient;
    sections: SectionsClient;
}

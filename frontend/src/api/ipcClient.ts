import { Member, Role, Group, ClubSection, Payment, Attachment, StorageStatus } from './types';
import { DataClient } from './dataClient';
import { LockInfo } from './ipcTypes';

function getApi() {
    if (!window.api) {
        throw new Error('window.api is not available. Make sure you are running in Electron with the preload script.');
    }
    return window.api;
}

export const ipcClient: DataClient = {
    club: {
        selectFolder: () => getApi().club.selectFolder(),
        getFolder: () => getApi().club.getFolder(),
        initFolder: async () => {
            await getApi().club.initFolder();
        },
    },

    members: {
        list: async (): Promise<Member[]> => {
            const result = await getApi().members.list();
            return result as Member[];
        },
        get: async (id: number): Promise<Member> => {
            const result = await getApi().members.get(id);
            return result as Member;
        },
        create: async (data: Omit<Member, 'id'>): Promise<Member> => {
            const result = await getApi().members.create(data);
            return result as Member;
        },
        update: async (data: Member): Promise<Member> => {
            const { id, ...patch } = data;
            const result = await getApi().members.update(id, patch);
            return result as Member;
        },
        delete: async (members: Member[]): Promise<void> => {
            const ids = members.map((m) => m.id);
            await getApi().members.delete(ids);
        },
        lock: (id: number, owner: string): Promise<{ acquired: boolean; lock: LockInfo | null }> =>
            getApi().members.lock(id, owner),
        unlock: (id: number, owner: string): Promise<{ released: boolean }> =>
            getApi().members.unlock(id, owner),
        getLock: (id: number): Promise<LockInfo | null> =>
            getApi().members.getLock(id),
    },

    roles: {
        list: async (): Promise<Role[]> => {
            const result = await getApi().roles.list();
            return result as Role[];
        },
        create: async (data: Omit<Role, 'id'>): Promise<Role> => {
            const result = await getApi().roles.create(data);
            return result as Role;
        },
        update: async (data: Role): Promise<Role> => {
            const result = await getApi().roles.update(data);
            return result as Role;
        },
        delete: async (data: Role): Promise<void> => {
            await getApi().roles.delete(data.id);
        },
    },

    groups: {
        list: async (): Promise<Group[]> => {
            const result = await getApi().groups.list();
            return result as Group[];
        },
        create: async (data: Omit<Group, 'id'>): Promise<Group> => {
            const result = await getApi().groups.create(data);
            return result as Group;
        },
        update: async (data: Group): Promise<Group> => {
            const result = await getApi().groups.update(data);
            return result as Group;
        },
        delete: async (data: Group): Promise<void> => {
            await getApi().groups.delete(data.id);
        },
    },

    sections: {
        list: async (): Promise<ClubSection[]> => {
            const result = await getApi().sections.list();
            return result as ClubSection[];
        },
        create: async (data: Omit<ClubSection, 'id'>): Promise<ClubSection> => {
            const result = await getApi().sections.create(data);
            return result as ClubSection;
        },
        update: async (data: ClubSection): Promise<ClubSection> => {
            const result = await getApi().sections.update(data);
            return result as ClubSection;
        },
        delete: async (data: ClubSection): Promise<void> => {
            await getApi().sections.delete(data.id);
        },
    },

    storage: {
        getStatus: (): Promise<StorageStatus> =>
            getApi().storage.getStatus() as Promise<StorageStatus>,
        requestEditMode: (): Promise<{ acquired: boolean; status: StorageStatus }> =>
            getApi().storage.requestEditMode() as Promise<{ acquired: boolean; status: StorageStatus }>,
        releaseEditMode: async (): Promise<void> => {
            await getApi().storage.releaseEditMode();
        },
        exportBackup: (): Promise<{ zipPath: string }> =>
            getApi().storage.exportBackup(),
    },

    payments: {
        list: async (): Promise<Payment[]> => {
            const result = await getApi().payments.list();
            return result as Payment[];
        },
        create: async (data: Omit<Payment, 'id' | 'createdAt'>): Promise<Payment> => {
            const result = await getApi().payments.create(data);
            return result as Payment;
        },
        update: async (data: Payment): Promise<Payment> => {
            const result = await getApi().payments.update(data);
            return result as Payment;
        },
        delete: async (id: string): Promise<void> => {
            await getApi().payments.delete(id);
        },
    },

    attachments: {
        add: async (data: {
            sourcePath: string;
            originalName: string;
            mimeType: string;
            paymentId?: string;
            memberId?: string;
        }): Promise<Attachment> => {
            const result = await getApi().attachments.add(data);
            return result as Attachment;
        },
        list: async (filter?: { paymentId?: string; memberId?: string }): Promise<Attachment[]> => {
            const result = await getApi().attachments.list(filter);
            return result as Attachment[];
        },
        open: async (id: string): Promise<void> => {
            await getApi().attachments.open(id);
        },
    },
};

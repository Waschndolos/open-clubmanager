import axios from 'axios';
import api from './api';

// Collaboration mode (docs/ux/collaboration-flows.md, /api/v2/collaboration).

export type AreaPermission = 'none' | 'read' | 'write';

export type Area = 'membersCore' | 'membersContact' | 'membersBank' | 'finance' | 'inventory' | 'admin';

export const AREAS: Area[] = ['membersCore', 'membersContact', 'membersBank', 'finance', 'inventory', 'admin'];

export type AreaPermissions = Record<Area, AreaPermission>;

export interface CollaborationStatus {
    mode: 'local' | 'joining' | 'member' | 'revoked';
    locked: boolean;
    clubName?: string;
    clubId?: string;
    storagePath?: string;
    storageError?: boolean;
    deviceId?: string;
    person?: string;
    deviceName?: string;
    role?: string;
    isAdmin: boolean;
    permissions: AreaPermissions;
    lastSync?: string;
    lastError?: string;
    syncing: boolean;
    pendingChanges: number;
    pendingRequests: number;
    conflicts: number;
    lostChanges: number;
    badKeys: number;
    /** Change streams of other devices waiting for a file, key or app update. */
    waitingStreams: number;
    /** Change streams stopped because a file failed verification. */
    quarantinedStreams: number;
    /** Received changes waiting for a row they reference. */
    waitingChanges: number;
    /** Readable areas without a verified starting snapshot yet. */
    incompleteAreas: number;
    /** Devices whose changes carried a clock far in the future. */
    clockWarnings: number;
}

export interface FolderClub {
    path: string;
    name: string;
}

export interface FolderInfo {
    path: string;
    writable: boolean;
    kind: 'sync' | 'network' | 'local';
    provider?: string;
    club?: FolderClub;
    clubsInside: FolderClub[];
}

export interface ClubInvite {
    id: string;
    person: string;
    role: string;
    createdAt: string;
    expiresAt: string;
    usedAt?: string;
    code?: string;
}

export interface ClubJoinRequest {
    deviceId: string;
    person: string;
    deviceName?: string;
    role?: string;
    error?: string;
}

export interface ClubDevice {
    deviceId: string;
    person: string;
    deviceName?: string;
    role: string;
    active: boolean;
    me: boolean;
}

export interface ClubRole {
    id: string;
    permissions: AreaPermissions;
}

export interface SyncConflict {
    table: string;
    rowId: string;
    column: string;
    value: string;
    keptRowId: string;
    detectedAt: string;
}

/** Error with the stable code the backend sends (e.g. LOCAL_DATA, WRONG_SAFETY_CODE). */
export class CollaborationError extends Error {
    constructor(public code: string, message: string) {
        super(message);
    }
}

async function call<T>(fn: () => Promise<{ data: T }>): Promise<T> {
    try {
        return (await fn()).data;
    } catch (e) {
        if (axios.isAxiosError(e)) {
            const data = e.response?.data as { code?: string; error?: string } | undefined;
            throw new CollaborationError(data?.code ?? 'NETWORK', data?.error ?? e.message);
        }
        throw e;
    }
}

export const getStatus = () => call(() => api.get<CollaborationStatus>('/collaboration/status'));
export const syncNow = () => call(() => api.post<CollaborationStatus>('/collaboration/sync'));
export const listFolders = () => call(() => api.get<FolderInfo[]>('/collaboration/folders'));
export const checkFolder = (path: string) => call(() => api.post<FolderInfo>('/collaboration/folders/check', { path }));
export const getLocalData = () => call(() => api.get<{ hasLocalData: boolean }>('/collaboration/local-data'));

export const createClub = (req: { folder: string; clubName: string; person: string; deviceName: string; password: string }) =>
    call(() => api.post<{ path: string; recoveryKit: string }>('/collaboration/club', req));

export const joinClub = (req: {
    folder: string; code: string; person: string; deviceName: string; password: string; replaceLocalData?: boolean;
}) => call(() => api.post<{ safetyCode: string }>('/collaboration/join', req));

export const getSafetyCode = () => call(() => api.get<{ safetyCode: string }>('/collaboration/join/safety-code'));

export const recoverClub = (req: {
    folder?: string; recoveryKit: string; person: string; deviceName: string; password: string; replaceLocalData?: boolean;
}) => call(() => api.post<CollaborationStatus>('/collaboration/recover', req));

export const listInvites = () => call(() => api.get<ClubInvite[]>('/collaboration/invites'));
export const createInvite = (person: string, role: string) =>
    call(() => api.post<ClubInvite>('/collaboration/invites', { person, role }));
export const withdrawInvite = (id: string) =>
    call(() => api.delete<void>(`/collaboration/invites/${encodeURIComponent(id)}`));

export const listRequests = () => call(() => api.get<ClubJoinRequest[]>('/collaboration/requests'));
export const approveRequest = (deviceId: string, safetyCode: string) =>
    call(() => api.post<void>(`/collaboration/requests/${deviceId}/approve`, { safetyCode }));
export const rejectRequest = (deviceId: string) =>
    call(() => api.post<void>(`/collaboration/requests/${deviceId}/reject`));

export const listDevices = () => call(() => api.get<ClubDevice[]>('/collaboration/devices'));
export const changeDeviceRole = (deviceId: string, role: string) =>
    call(() => api.put<void>(`/collaboration/devices/${deviceId}/role`, { role }));
export const revokeDevice = (deviceId: string) =>
    call(() => api.post<void>(`/collaboration/devices/${deviceId}/revoke`));

export const listRoles = () => call(() => api.get<ClubRole[]>('/collaboration/roles'));
export const listConflicts = () => call(() => api.get<SyncConflict[]>('/collaboration/conflicts'));

/** Formats a 6-digit safety code for reading aloud: "482 913". */
export function formatSafetyCode(code: string): string {
    return code.length === 6 ? `${code.slice(0, 3)} ${code.slice(3)}` : code;
}

import { isAxiosError } from 'axios';
import api from './api';
import { HonorMember } from './honors';
import { EventCategory, NamedRef } from './events';

export type AttendanceStatus = 'present' | 'excused' | 'unexcused';
export const ATTENDANCE_STATES: AttendanceStatus[] = ['present', 'excused', 'unexcused'];

export interface AttendanceOccurrence {
    eventId: string;
    /** Local start date (YYYY-MM-DD). */
    occurrenceDate: string;
    title: string;
    category: EventCategory;
    start: string;
    end: string;
    allDay: boolean;
    groups: NamedRef[];
    sections: NamedRef[];
    roles: NamedRef[];
    present: number;
    excused: number;
    unexcused: number;
}

export interface AttendanceMember extends HonorMember {
    status?: AttendanceStatus;
    /** False for guests recorded although they are not in the event's audience. */
    inAudience: boolean;
}

export interface AttendanceSheet {
    occurrence: AttendanceOccurrence;
    members: AttendanceMember[];
}

export interface AttendanceEntry {
    memberId: string;
    /** null clears the entry. */
    status: AttendanceStatus | null;
}

export interface MemberAttendance {
    member: HonorMember;
    present: number;
    excused: number;
    unexcused: number;
    /** Share of recorded sessions attended, 0..1. */
    rate: number;
}

export interface AttendanceStats {
    members: MemberAttendance[];
    occurrences: AttendanceOccurrence[];
}

export interface AttendanceFilter {
    groupId?: string;
    sectionId?: string;
}

export async function fetchAttendanceOccurrences(from: string, to: string, filter: AttendanceFilter = {}): Promise<AttendanceOccurrence[]> {
    return (await api.get<AttendanceOccurrence[]>('/attendance/occurrences', { params: { from, to, ...filter } })).data;
}

export async function fetchAttendanceSheet(eventId: string, date: string): Promise<AttendanceSheet> {
    return (await api.get<AttendanceSheet>('/attendance/sheet', { params: { eventId, date } })).data;
}

export async function fetchAttendanceStats(from: string, to: string, filter: AttendanceFilter = {}): Promise<AttendanceStats> {
    return (await api.get<AttendanceStats>('/attendance/stats', { params: { from, to, ...filter } })).data;
}

export async function downloadAttendance(from: string, to: string, format: 'csv' | 'pdf', filter: AttendanceFilter = {}): Promise<void> {
    const res = await api.get('/attendance/export', { params: { from, to, format, ...filter }, responseType: 'blob' });
    const href = URL.createObjectURL(new Blob([res.data], { type: format === 'pdf' ? 'application/pdf' : 'text/csv' }));
    const anchor = document.createElement('a');
    anchor.href = href;
    anchor.download = `attendance-${from}-${to}.${format}`;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(href);
}

// ── Offline queue ────────────────────────────────────────────────────────────
// When the server cannot be reached (e.g. on a phone at the training ground
// without network), changes are kept in localStorage and sent later. Entries
// are idempotent, so resending is safe; newer changes for the same member and
// occurrence replace older ones.

interface QueuedChange {
    eventId: string;
    date: string;
    entries: AttendanceEntry[];
}

const QUEUE_KEY = 'attendanceQueue';

function readQueue(): QueuedChange[] {
    try {
        const raw = localStorage.getItem(QUEUE_KEY);
        return raw ? (JSON.parse(raw) as QueuedChange[]) : [];
    } catch {
        return [];
    }
}

function writeQueue(queue: QueuedChange[]) {
    try {
        if (queue.length === 0) localStorage.removeItem(QUEUE_KEY);
        else localStorage.setItem(QUEUE_KEY, JSON.stringify(queue));
    } catch {
        // Storage unavailable (private mode): the change is lost, as before.
    }
    window.dispatchEvent(new Event('attendance-queue'));
}

function enqueue(change: QueuedChange) {
    const queue = readQueue();
    let target = queue.find((q) => q.eventId === change.eventId && q.date === change.date);
    if (!target) {
        target = { eventId: change.eventId, date: change.date, entries: [] };
        queue.push(target);
    }
    for (const e of change.entries) {
        target.entries = target.entries.filter((x) => x.memberId !== e.memberId);
        target.entries.push(e);
    }
    writeQueue(queue);
}

/** Number of occurrences with changes waiting to be sent. */
export function pendingAttendance(): number {
    return readQueue().length;
}

/** Changes queued for an occurrence, to show them on its sheet. */
export function queuedEntries(eventId: string, date: string): AttendanceEntry[] {
    return readQueue().find((q) => q.eventId === eventId && q.date === date)?.entries ?? [];
}

const isOffline = (e: unknown) => isAxiosError(e) && !e.response;

/**
 * Saves entries for an occurrence. Returns the new sheet, or null when the
 * server was unreachable and the change was queued for later.
 */
export async function saveAttendance(eventId: string, date: string, entries: AttendanceEntry[]): Promise<AttendanceSheet | null> {
    try {
        return (await api.put<AttendanceSheet>('/attendance/sheet', { eventId, date, entries })).data;
    } catch (e) {
        if (isOffline(e)) {
            enqueue({ eventId, date, entries });
            return null;
        }
        throw e;
    }
}

let flushing = false;

/**
 * Sends queued changes. Changes the server rejects for good (e.g. a deleted
 * event) are dropped so that they do not block the queue. Returns the number
 * sent.
 */
export async function flushAttendanceQueue(): Promise<number> {
    if (flushing) return 0;
    flushing = true;
    let sent = 0;
    try {
        for (const change of readQueue()) {
            try {
                await api.put('/attendance/sheet', change);
                sent++;
            } catch (e) {
                // Drop only changes the server rejects for good (deleted event,
                // invalid data); keep them while offline, signed out or failing.
                const status = isAxiosError(e) ? e.response?.status : undefined;
                if (status !== 400 && status !== 404) break;
            }
            // Remove what was sent, but keep entries changed again meanwhile.
            writeQueue(readQueue()
                .map((q) => q.eventId === change.eventId && q.date === change.date
                    ? { ...q, entries: q.entries.filter((e) => !change.entries.some((s) => s.memberId === e.memberId && s.status === e.status)) }
                    : q)
                .filter((q) => q.entries.length > 0));
        }
    } finally {
        flushing = false;
    }
    return sent;
}

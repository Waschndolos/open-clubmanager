import api from './api';

export type EventCategory = 'training' | 'festival' | 'meeting' | 'competition' | 'other';
export type EventRecurrence = 'none' | 'weekly' | 'biweekly' | 'monthly';

export const EVENT_CATEGORIES: EventCategory[] = ['training', 'festival', 'meeting', 'competition', 'other'];
export const EVENT_RECURRENCES: EventRecurrence[] = ['none', 'weekly', 'biweekly', 'monthly'];

export const CATEGORY_COLOR: Record<EventCategory, 'primary' | 'secondary' | 'info' | 'warning' | 'default'> = {
    training: 'primary',
    festival: 'secondary',
    meeting: 'info',
    competition: 'warning',
    other: 'default',
};

export interface NamedRef {
    id: string;
    name: string;
}

export interface ClubEvent {
    id: string;
    title: string;
    description?: string | null;
    location?: string | null;
    category: EventCategory;
    /** ISO date-time; for all-day events `end` is the midnight after the last day. */
    start: string;
    end: string;
    allDay: boolean;
    timeZone: string;
    recurrence: EventRecurrence;
    /** YYYY-MM-DD */
    recurrenceUntil?: string | null;
    exceptions: string[];
    groups: NamedRef[];
    sections: NamedRef[];
    roles: NamedRef[];
}

export interface EventInput {
    title: string;
    description?: string;
    location?: string;
    category: EventCategory;
    start: string;
    end: string;
    allDay: boolean;
    timeZone: string;
    recurrence: EventRecurrence;
    recurrenceUntil?: string;
    groupIds: string[];
    sectionIds: string[];
    roleIds: string[];
}

export interface EventOccurrence {
    eventId: string;
    /** Local start date (YYYY-MM-DD) identifying the occurrence. */
    occurrenceDate: string;
    title: string;
    description?: string | null;
    location?: string | null;
    category: EventCategory;
    start: string;
    end: string;
    allDay: boolean;
    recurring: boolean;
    groups: NamedRef[];
    sections: NamedRef[];
    roles: NamedRef[];
}

/** Restricts events to a group, section or role; club-wide events always match. */
export interface AudienceFilter {
    groupId?: string;
    sectionId?: string;
    roleId?: string;
}

/** Occurrences overlapping the local days from..to (YYYY-MM-DD, inclusive). */
export async function fetchOccurrences(from: string, to: string, filter: AudienceFilter = {}): Promise<EventOccurrence[]> {
    return (await api.get<EventOccurrence[]>('/events', { params: { from, to, ...filter } })).data;
}

export async function fetchEvent(id: string): Promise<ClubEvent> {
    return (await api.get<ClubEvent>(`/events/${encodeURIComponent(id)}`)).data;
}

export async function createEvent(data: EventInput): Promise<ClubEvent> {
    return (await api.post<ClubEvent>('/events', data)).data;
}

export async function updateEvent(id: string, data: EventInput): Promise<ClubEvent> {
    return (await api.put<ClubEvent>(`/events/${encodeURIComponent(id)}`, data)).data;
}

export async function deleteEvent(id: string): Promise<void> {
    await api.delete(`/events/${encodeURIComponent(id)}`);
}

/** Removes one occurrence of a recurring event. */
export async function excludeOccurrence(id: string, date: string): Promise<void> {
    await api.post(`/events/${encodeURIComponent(id)}/exceptions`, { date });
}

async function download(url: string, params: Record<string, string | undefined>, filename: string) {
    const res = await api.get(url, { params, responseType: 'blob' });
    const href = URL.createObjectURL(new Blob([res.data], { type: 'text/calendar' }));
    const anchor = document.createElement('a');
    anchor.href = href;
    anchor.download = filename;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(href);
}

/** Downloads one occurrence (or the whole series without date) as .ics. */
export async function downloadEventIcs(id: string, title: string, date?: string): Promise<void> {
    const name = title.toLowerCase().replace(/[^a-z0-9äöüß]+/g, '-').replace(/^-|-$/g, '') || 'event';
    await download(`/events/${encodeURIComponent(id)}/ics`, { date }, `${name}.ics`);
}

/** Downloads the club calendar as .ics, optionally for one audience. */
export async function downloadCalendarIcs(filter: AudienceFilter = {}): Promise<void> {
    await download('/calendar.ics', { ...filter }, 'calendar.ics');
}

/** The local date of d as YYYY-MM-DD. */
export function ymd(d: Date): string {
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

export function addDays(d: Date, days: number): Date {
    const r = new Date(d);
    r.setDate(r.getDate() + days);
    return r;
}

/** Monday of the week containing d, at local midnight. */
export function startOfWeek(d: Date): Date {
    const r = new Date(d.getFullYear(), d.getMonth(), d.getDate());
    r.setDate(r.getDate() - ((r.getDay() + 6) % 7));
    return r;
}

export function localTimeZone(): string {
    try {
        return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
    } catch {
        return 'UTC';
    }
}

/** Whether the occurrence covers (part of) the local day. */
export function coversDay(o: EventOccurrence, day: Date): boolean {
    const dayStart = new Date(day.getFullYear(), day.getMonth(), day.getDate());
    const dayEnd = addDays(dayStart, 1);
    const start = new Date(o.start);
    const end = new Date(o.end);
    // A zero-length event still shows on its start day.
    return start < dayEnd && (end > dayStart || (end.getTime() === start.getTime() && start >= dayStart));
}

import { Event, EventAttendee, EventAttendanceStatus, EventType } from './types';
import { getDataClient } from './clientFactory';

export async function fetchEvents(filters?: { startDateFrom?: string; startDateTo?: string; type?: EventType }): Promise<Event[]> {
    return getDataClient().events.list(filters);
}

export async function fetchEvent(id: number): Promise<Event> {
    return getDataClient().events.get(id);
}

export async function createEvent(data: Omit<Event, 'id' | 'createdAt' | 'updatedAt'>): Promise<Event> {
    return getDataClient().events.create(data);
}

export async function updateEvent(id: number, data: Partial<Omit<Event, 'id' | 'createdAt' | 'updatedAt'>>): Promise<Event> {
    return getDataClient().events.update(id, data);
}

export async function deleteEvent(id: number): Promise<void> {
    return getDataClient().events.delete(id);
}

export async function fetchEventAttendees(eventId: number): Promise<EventAttendee[]> {
    return getDataClient().events.listAttendees(eventId);
}

export async function upsertEventAttendees(eventId: number, attendees: Array<{ memberId: number; status?: EventAttendanceStatus }>): Promise<void> {
    return getDataClient().events.upsertAttendees(eventId, attendees);
}

export async function removeEventAttendee(eventId: number, memberId: number): Promise<void> {
    return getDataClient().events.removeAttendee(eventId, memberId);
}

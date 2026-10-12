import api from './api';
import { HonorMember } from './honors';

export type TaskStatus = 'open' | 'in_progress' | 'done';
export type TaskRecurrence = 'none' | 'weekly' | 'monthly' | 'yearly';
export type TaskFilter = 'all' | 'mine' | 'overdue' | 'done';

export interface Task {
    id: string;
    title: string;
    description?: string;
    dueDate?: string;
    status: TaskStatus;
    recurrence: TaskRecurrence;
    assigneeMember?: HonorMember;
    assigneeRole?: { id: string; name: string };
    linkedMember?: HonorMember;
    overdue: boolean;
}

export interface TaskInput {
    title: string;
    description?: string;
    dueDate?: string;
    status: TaskStatus;
    recurrence: TaskRecurrence;
    assigneeMemberId?: string;
    assigneeRoleId?: string;
    linkedMemberId?: string;
}

export async function fetchTasks(filter: TaskFilter = 'all'): Promise<Task[]> {
    return (await api.get<Task[]>('/tasks', { params: { filter } })).data;
}

export async function createTask(data: TaskInput): Promise<Task> {
    return (await api.post<Task>('/tasks', data)).data;
}

export async function updateTask(id: string, data: TaskInput): Promise<Task> {
    return (await api.put<Task>(`/tasks/${encodeURIComponent(id)}`, data)).data;
}

export async function deleteTask(id: string): Promise<void> {
    await api.delete(`/tasks/${encodeURIComponent(id)}`);
}

/** The writable fields of an existing task, for changing a single field. */
export function taskToInput(t: Task): TaskInput {
    return {
        title: t.title, description: t.description, dueDate: t.dueDate, status: t.status, recurrence: t.recurrence,
        assigneeMemberId: t.assigneeMember?.id, assigneeRoleId: t.assigneeRole?.id, linkedMemberId: t.linkedMember?.id,
    };
}

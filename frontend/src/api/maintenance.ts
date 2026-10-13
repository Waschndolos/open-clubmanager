import api from './api';

export type ItemCondition = 'ready' | 'damaged' | 'in_repair' | 'retired';
export type DueState = 'none' | 'ok' | 'due' | 'overdue';

export const CONDITIONS: ItemCondition[] = ['ready', 'damaged', 'in_repair', 'retired'];

export interface ItemStatus {
    condition: ItemCondition;
    intervalMonths?: number;
    /** YYYY-MM-DD */
    nextDue?: string;
}

export interface MaintenanceState extends ItemStatus {
    itemId: string;
    name: string;
    category: string;
    location: string;
    lastMaintenance?: string;
    due: DueState;
}

export interface Maintenance {
    id: string;
    itemId: string;
    /** YYYY-MM-DD */
    date: string;
    performedBy?: string;
    note?: string;
    cost?: number;
    transactionId?: string;
}

export interface MaintenanceInput {
    date: string;
    performedBy?: string;
    note?: string;
    cost?: number;
    condition?: ItemCondition;
    /** Books the cost as an expense in the cash book. */
    bookExpense?: { description: string; category?: string };
}

/** Condition, schedule and due state of every item. */
export async function fetchMaintenanceStates(warnDays = 30): Promise<MaintenanceState[]> {
    return (await api.get<MaintenanceState[]>('/inventory/maintenance', { params: { warnDays } })).data;
}

export async function setItemStatus(itemId: string, status: ItemStatus): Promise<void> {
    await api.put(`/inventory/items/${encodeURIComponent(itemId)}/status`, status);
}

export async function fetchItemMaintenance(itemId: string): Promise<Maintenance[]> {
    return (await api.get<Maintenance[]>(`/inventory/items/${encodeURIComponent(itemId)}/maintenance`)).data;
}

export async function recordMaintenance(itemId: string, data: MaintenanceInput): Promise<Maintenance> {
    return (await api.post<Maintenance>(`/inventory/items/${encodeURIComponent(itemId)}/maintenance`, data)).data;
}

export async function deleteMaintenance(id: string): Promise<void> {
    await api.delete(`/inventory/maintenance/${encodeURIComponent(id)}`);
}

export const CONDITION_COLOR: Record<ItemCondition, 'success' | 'error' | 'warning' | 'default'> = {
    ready: 'success',
    damaged: 'error',
    in_repair: 'warning',
    retired: 'default',
};

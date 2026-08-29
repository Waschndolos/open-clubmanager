import api from './api';
import { InventoryItem, InventoryLoan } from './types';

export async function fetchInventoryItems(): Promise<InventoryItem[]> {
    const res = await api.get<InventoryItem[]>('/inventory/items');
    return res.data;
}

export async function createInventoryItem(
    data: Omit<InventoryItem, 'id' | 'createdAt' | 'updatedAt'>
): Promise<InventoryItem> {
    const res = await api.post<InventoryItem>('/inventory/items', data);
    return res.data;
}

export async function updateInventoryItem(
    id: number,
    data: Partial<Omit<InventoryItem, 'id' | 'createdAt' | 'updatedAt'>>
): Promise<InventoryItem> {
    const res = await api.put<InventoryItem>(`/inventory/items/${id}`, data);
    return res.data;
}

export async function deleteInventoryItem(id: number): Promise<void> {
    await api.delete(`/inventory/items/${id}`);
}

export async function fetchInventoryLoans(): Promise<InventoryLoan[]> {
    const res = await api.get<InventoryLoan[]>('/inventory/loans');
    return res.data;
}

export async function createInventoryLoan(
    data: Omit<InventoryLoan, 'id' | 'createdAt'>
): Promise<InventoryLoan> {
    const res = await api.post<InventoryLoan>('/inventory/loans', data);
    return res.data;
}

export async function updateInventoryLoan(
    id: number,
    data: Partial<Omit<InventoryLoan, 'id' | 'createdAt'>>
): Promise<InventoryLoan> {
    const res = await api.put<InventoryLoan>(`/inventory/loans/${id}`, data);
    return res.data;
}

export async function deleteInventoryLoan(id: number): Promise<void> {
    await api.delete(`/inventory/loans/${id}`);
}

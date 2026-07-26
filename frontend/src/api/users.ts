import api from './api';
import { AppUser, AppRole } from './types';

export async function listAppUsers(): Promise<AppUser[]> {
    const res = await api.get<AppUser[]>('/users');
    return res.data;
}

export async function createAppUser(email: string, password: string, appRole: AppRole): Promise<AppUser> {
    const res = await api.post<AppUser>('/users', { email, password, appRole });
    return res.data;
}

export async function updateAppUser(id: number, email?: string, appRole?: AppRole): Promise<AppUser> {
    const res = await api.put<AppUser>(`/users/${id}`, { email, appRole });
    return res.data;
}

export async function deleteAppUser(id: number): Promise<void> {
    await api.delete(`/users/${id}`);
}

export async function changePassword(newPassword: string, currentPassword?: string): Promise<void> {
    await api.post('/auth/change-password', { newPassword, currentPassword });
}

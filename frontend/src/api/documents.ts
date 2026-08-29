import api from './api';
import { ClubDocument } from './types';

export type UploadDocumentPayload = {
    title: string;
    category: string;
    description?: string;
    file: File;
};

export async function fetchDocuments(category?: string): Promise<ClubDocument[]> {
    const res = await api.get<ClubDocument[]>('/documents', {
        params: category ? { category } : undefined,
    });
    return res.data;
}

export async function uploadDocument(payload: UploadDocumentPayload): Promise<ClubDocument> {
    const formData = new FormData();
    formData.append('title', payload.title);
    formData.append('category', payload.category);
    if (payload.description) {
        formData.append('description', payload.description);
    }
    formData.append('file', payload.file);

    const res = await api.post<ClubDocument>('/documents', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
    });
    return res.data;
}

export async function updateDocument(
    id: number,
    payload: Partial<Pick<ClubDocument, 'title' | 'description' | 'category'>>
): Promise<ClubDocument> {
    const res = await api.put<ClubDocument>(`/documents/${id}`, payload);
    return res.data;
}

export async function deleteDocument(id: number): Promise<void> {
    await api.delete(`/documents/${id}`);
}

export async function downloadDocument(id: number, filename: string): Promise<void> {
    const res = await api.get<Blob>(`/documents/${id}`, { responseType: 'blob' });
    const objectUrl = window.URL.createObjectURL(res.data);
    const anchor = window.document.createElement('a');
    anchor.href = objectUrl;
    anchor.download = filename;
    window.document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    window.setTimeout(() => window.URL.revokeObjectURL(objectUrl), 60_000);
}

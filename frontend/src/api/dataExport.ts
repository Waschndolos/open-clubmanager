import api from './api';
import { BACKEND_URL } from './baseUrl';

export type DataExportFormat = 'pdf' | 'json';

/** Downloads the GDPR data export (Art. 15/20) of one member. */
export async function downloadMemberDataExport(memberId: string, format: DataExportFormat): Promise<void> {
    const response = await api.get(`${BACKEND_URL}/members/${encodeURIComponent(memberId)}/data-export`, {
        params: { format },
        responseType: 'blob',
    });
    const type = format === 'pdf' ? 'application/pdf' : 'application/json';
    const url = URL.createObjectURL(new Blob([response.data], { type }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `data-export-${memberId}.${format}`;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(url);
}

function normalizeOrigin(origin: string): string {
    return origin.endsWith('/') ? origin.slice(0, -1) : origin;
}

const isTauri = typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window;
const browserOrigin = '';
const tauriOrigin = 'http://127.0.0.1:3001';

export const BACKEND_ORIGIN = normalizeOrigin(
    import.meta.env.VITE_BACKEND_ORIGIN ?? (isTauri ? tauriOrigin : browserOrigin)
);
export const BACKEND_URL = `${BACKEND_ORIGIN}/api/v2`;
export const BACKEND_V2_URL = BACKEND_URL;

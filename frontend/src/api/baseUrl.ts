function normalizeOrigin(origin: string): string {
    return origin.endsWith('/') ? origin.slice(0, -1) : origin;
}

const isTauri = typeof import.meta.env.TAURI_ENV_PLATFORM === 'string';
const browserRelativeOrigin = '';
const tauriOrigin = 'http://127.0.0.1:3001';

export const BACKEND_ORIGIN = normalizeOrigin(
    import.meta.env.VITE_BACKEND_ORIGIN ?? (isTauri ? tauriOrigin : browserRelativeOrigin)
);
export const BACKEND_URL = `${BACKEND_ORIGIN}/api/v2`;
export const BACKEND_V2_URL = BACKEND_URL;

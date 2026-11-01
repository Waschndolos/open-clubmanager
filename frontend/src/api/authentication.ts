import { BACKEND_V2_URL } from './baseUrl';

const BASE_URL = `${BACKEND_V2_URL}/auth`;

/**
 * Refresh tokens: a browser page on the same site as the backend (the dev
 * setup) gets the refresh token as an httpOnly cookie that scripts cannot read.
 * The desktop app serves its pages from tauri://localhost or
 * http://tauri.localhost, a different site than the backend, where such a
 * cookie is never sent. It asks for the token in the response body instead and
 * keeps it in sessionStorage: it survives a reload, but not closing the app.
 */
function backendIsSameSite(): boolean {
    try {
        const backend = new URL(BASE_URL);
        return backend.protocol === window.location.protocol && backend.hostname === window.location.hostname;
    } catch {
        return false;
    }
}

const REFRESH_IN_BODY = !backendIsSameSite();
const REFRESH_STORAGE_KEY = "ocm.refreshToken";

function storedRefreshToken(): string | null {
    try {
        return sessionStorage.getItem(REFRESH_STORAGE_KEY);
    } catch {
        return null;
    }
}

function storeRefreshToken(token: string | null | undefined): void {
    try {
        if (token) sessionStorage.setItem(REFRESH_STORAGE_KEY, token);
        else sessionStorage.removeItem(REFRESH_STORAGE_KEY);
    } catch {
        // Without storage the login just ends with the access token.
    }
}

/** Headers that carry the refresh token for clients without the cookie. */
function refreshHeaders(): Record<string, string> {
    const token = REFRESH_IN_BODY ? storedRefreshToken() : null;
    return token ? { "X-Refresh-Token": token } : {};
}

type TokenResponse = { accessToken: string; refreshToken?: string };

export async function login(email: string, password: string): Promise<TokenResponse> {
    const res = await fetch(`${BASE_URL}/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password, refreshTokenInBody: REFRESH_IN_BODY }),
        credentials: "include", // receives the httpOnly refresh cookie
    });

    // status lets the caller tell a lockout (429) from wrong credentials (401).
    if (!res.ok) throw Object.assign(new Error("Login failed"), { status: res.status });

    const data = (await res.json()) as TokenResponse;
    if (REFRESH_IN_BODY) storeRefreshToken(data.refreshToken);
    return data;
}

/**
 * Exchanges the refresh token for a new access token. The refresh token is
 * rotated on every call, so callers must not run two at once (see
 * refreshSession in api.ts).
 */
export async function refreshAccessToken(): Promise<TokenResponse> {
    const headers = refreshHeaders();
    if (REFRESH_IN_BODY && !headers["X-Refresh-Token"]) {
        throw new Error("No refresh token");
    }
    const res = await fetch(`${BASE_URL}/refresh-token`, {
        method: "POST",
        headers,
        credentials: "include", // sends the httpOnly refresh cookie
    });

    if (!res.ok) {
        if (REFRESH_IN_BODY && res.status === 401) storeRefreshToken(null);
        throw new Error("Refresh token invalid");
    }

    const data = (await res.json()) as TokenResponse;
    if (REFRESH_IN_BODY && data.refreshToken) storeRefreshToken(data.refreshToken);
    return data;
}

/** Revokes the refresh token on the server and forgets it locally. */
export async function logout(): Promise<void> {
    const headers = refreshHeaders();
    storeRefreshToken(null);
    try {
        await fetch(`${BASE_URL}/logout`, { method: "POST", headers, credentials: "include" });
    } catch {
        // The server may be gone; the local session is cleared regardless.
    }
}

export async function getProfile(token: string) {
    const res = await fetch(`${BASE_URL}/profile`, {
        headers: { Authorization: `Bearer ${token}` },
    });

    if (!res.ok) throw new Error("Unauthorized");

    return res.json(); // { email }
}

export async function forgotPassword(email: string) {
    const res = await fetch(`${BASE_URL}/forgot-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
    });

    if (!res.ok) throw new Error("Request failed");

    return res.json();
}

export async function resetPassword(token: string, newPassword: string) {
    const res = await fetch(`${BASE_URL}/reset-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token, newPassword }),
    });

    if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error((data as { error?: string }).error ?? "Reset failed");
    }

    return res.json();
}

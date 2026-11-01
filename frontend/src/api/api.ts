import axios, {
    AxiosError,
    AxiosInstance,
    InternalAxiosRequestConfig,
} from "axios";
import { BACKEND_ORIGIN, BACKEND_URL, BACKEND_V2_URL } from './baseUrl';

export { BACKEND_ORIGIN, BACKEND_URL, BACKEND_V2_URL };

interface RetryAxiosRequestConfig extends InternalAxiosRequestConfig {
    _retry?: boolean;
}

const api: AxiosInstance = axios.create({
    baseURL: BACKEND_URL,
    withCredentials: true,
});

let accessToken: string | null = null;

type TokenListener = (token: string | null) => void;
const tokenListeners = new Set<TokenListener>();

/** Sets the access token used for API requests and tells the AuthContext. */
export const setAccessToken = (token: string | null): void => {
    if (token === accessToken) return;
    accessToken = token;
    tokenListeners.forEach((listener) => listener(token));
};

/** Subscribes to access token changes (login, refresh, logout). */
export const onAccessTokenChange = (listener: TokenListener): (() => void) => {
    tokenListeners.add(listener);
    return () => {
        tokenListeners.delete(listener);
    };
};

let sessionExpiredHandler: (() => void) | null = null;

/** Registers what happens when the session cannot be refreshed any more. */
export const onSessionExpired = (handler: (() => void) | null): void => {
    sessionExpiredHandler = handler;
};

let refreshing: Promise<string> | null = null;

/**
 * Gets a new access token with the refresh token. Concurrent callers share
 * one request: the server rotates the refresh token on every refresh, so a
 * second parallel refresh would present an already replaced token.
 */
export const refreshSession = (): Promise<string> => {
    if (!refreshing) {
        refreshing = import('./authentication')
            .then(({ refreshAccessToken }) => refreshAccessToken())
            .then(({ accessToken: token }) => {
                setAccessToken(token);
                return token;
            })
            .finally(() => {
                refreshing = null;
            });
    }
    return refreshing;
};

api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
    if (accessToken) {
        config.headers = config.headers ?? {};
        config.headers.Authorization = `Bearer ${accessToken}`;
    }
    return config;
});

api.interceptors.response.use(
    (response) => response,
    async (error: AxiosError) => {
        const originalRequest = error.config as RetryAxiosRequestConfig | undefined;

        // Only a request sent with a token belongs to a session that a
        // refresh can extend; each request is retried at most once.
        if (
            !originalRequest ||
            error.response?.status !== 401 ||
            originalRequest._retry ||
            !originalRequest.headers?.Authorization
        ) {
            return Promise.reject(error);
        }
        originalRequest._retry = true;

        const sentToken = String(originalRequest.headers.Authorization).replace(/^Bearer /, '');
        let token: string;
        try {
            // Another request may have refreshed while this one was in flight.
            token = accessToken && accessToken !== sentToken ? accessToken : await refreshSession();
        } catch {
            if (accessToken) {
                sessionExpiredHandler?.();
                setAccessToken(null);
            }
            return Promise.reject(error);
        }
        originalRequest.headers.Authorization = `Bearer ${token}`;
        return api(originalRequest);
    }
);

export default api;

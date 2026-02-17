import axios, {
    AxiosError,
    AxiosInstance,
    InternalAxiosRequestConfig,
} from "axios";
import { refreshAccessToken } from "./authentication";

export const BACKEND_URL = "http://localhost:3001/api";

interface RetryAxiosRequestConfig extends InternalAxiosRequestConfig {
    _retry?: boolean;
}

const api: AxiosInstance = axios.create({
    baseURL: BACKEND_URL,
    withCredentials: true,
});

let accessToken: string | null = null;
let isRefreshing = false;

type QueuePromise = {
    resolve: (token: string | null) => void;
    reject: (error: unknown) => void;
};

let failedQueue: QueuePromise[] = [];

const processQueue = (error: unknown, token: string | null = null): void => {
    failedQueue.forEach(({ resolve, reject }) => {
        if (error) {
            reject(error);
        } else {
            resolve(token);
        }
    });

    failedQueue = [];
};

api.interceptors.request.use(
    (config: InternalAxiosRequestConfig) => {
    if (accessToken) {
            config.headers = config.headers ?? {};
        config.headers.Authorization = `Bearer ${accessToken}`;
    }
    return config;
    }
);

api.interceptors.response.use(
    (response) => response,
    async (error: AxiosError) => {
        const originalRequest = error.config as RetryAxiosRequestConfig;

        if (!originalRequest) {
            return Promise.reject(error);
        }

        if (error.response?.status === 401 && !originalRequest._retry) {
            if (isRefreshing) {
                return new Promise<string | null>((resolve, reject) => {
                    failedQueue.push({ resolve, reject });
                })
                    .then((token) => {
                        if (token) {
                        originalRequest.headers.Authorization = `Bearer ${token}`;
                        }
                        return api(originalRequest);
                    })
                    .catch((err) => Promise.reject(err));
            }

            originalRequest._retry = true;
            isRefreshing = true;

                try {
                    const data = await refreshAccessToken();
                    accessToken = data.accessToken;

                    processQueue(null, accessToken);

                if (accessToken) {
                    originalRequest.headers.Authorization = `Bearer ${accessToken}`;
                }

                return api(originalRequest);
                } catch (err) {
                    processQueue(err, null);
                return Promise.reject(err);
                } finally {
                    isRefreshing = false;
                }
        }

        return Promise.reject(error);
    }
);

export const setAccessToken = (token: string | null): void => {
    accessToken = token;
};

export default api;

export {};

declare global {
    interface Window {
        electronAPI: {
            getSetting: (key: string) => unknown;
            setSetting: (key: string, value: unknown) => void;
        };
        apppreference?: {
            get<T = unknown>(key: string): T;
            set(key: string, value: unknown): void;
        };
        userpreference?: {
            get<T = unknown>(userId: string, key: string): T;
            set(userId: string, key: string, value: unknown): void;
        };
    }
}

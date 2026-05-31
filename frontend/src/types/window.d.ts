export {};

declare global {
    interface Window {
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

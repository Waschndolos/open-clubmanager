
type AppPreferenceAPI = {
    get: (key: string) => unknown;
    set: (key: string, value: unknown) => void;
};

type UserPreferenceAPI = {
    get: (userId: string, key: string) => unknown;
    set: (userId: string, key: string, value: unknown) => void;
};


const fallbackAppPreference: AppPreferenceAPI = {
    get(key) {
        const val = localStorage.getItem(`apppreference:${key}`);
        try {
            return JSON.parse(val ?? "null");
        } catch {
            return val;
        }
    },
    set(key, value) {
        localStorage.setItem(`apppreference:${key}`, JSON.stringify(value));
    }
};

const fallbackUserPreference: UserPreferenceAPI = {
    get(userId, key) {
        const val = localStorage.getItem(`userpreference:${userId}:${key}`);
        try {
            return JSON.parse(val ?? "null");
        } catch {
            return val;
        }
    },
    set(userId, key, value) {
        localStorage.setItem(`userpreference:${userId}:${key}`, JSON.stringify(value));
    }
};


export const apppreference: AppPreferenceAPI =
    window.apppreference ?? fallbackAppPreference;

export const userpreference: UserPreferenceAPI =
    window.userpreference ?? fallbackUserPreference;

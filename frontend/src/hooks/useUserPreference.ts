import { useCallback } from "react";
import { isAxiosError } from "axios";
import api from '../api/api';

export function useUserPreference() {

    const getPreference = useCallback(async (key: string) => {
        try {
            const res = await api.get<{ value: unknown }>(`/preferences/${key}`);
            // 204: nothing stored yet, the caller uses its defaults.
            return res.status === 204 ? null : res.data.value;

        } catch (err) {
            if (isAxiosError(err) && err.response?.status === 404) return null;
            console.error("Failed to fetch preference:", err);
            return null;
        }
    }, []);

    const setPreference = useCallback(async (key: string, value: unknown) => {
        try {
            await api.post('/preferences', { key, value });
        } catch (err) {
            console.error("Failed to save preference:", err);
        }
    }, []);

    return { getPreference, setPreference };
}

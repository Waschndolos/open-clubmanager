import { useCallback } from "react";

const API_URL = "http://localhost:3001/api/preferences";

export function useUserPreference() {

    const getPreference = useCallback(async (key: string) => {
        try {
            const res = await fetch(`${API_URL}/${key}`, {
                credentials: "include"
            });

            if (!res.ok) {
                if (res.status === 404) return null;
                throw new Error(`HTTP ${res.status}`);
            }

            const data = await res.json();
            return data.value;

        } catch (err) {
            console.error("Failed to fetch preference:", err);
            return null;
        }
    }, []);

    const setPreference = useCallback(async (key: string, value: unknown) => {
        try {
            const res = await fetch(API_URL, {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ key, value }),
            });

            if (!res.ok) {
                throw new Error(`HTTP ${res.status}`);
            }

        } catch (err) {
            console.error("Failed to save preference:", err);
        }
    }, []);

    return { getPreference, setPreference };
}

import { useCallback } from "react";

const API_URL = "http://localhost:3001/api/preference"; // ggf. .env

export function useUserPreference() {
    const getPreference = useCallback(async (key: string) => {
        try {
            const res = await fetch(`${API_URL}/${key}`);
            if (!res.ok) return null;
            const data = await res.json();
            return data.value;
        } catch (err) {
            console.error("Failed to fetch preference:", err);
            return null;
        }
    }, []);

    const setPreference = useCallback(async (key: string, value: unknown) => {
        try {
            await fetch(API_URL, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ key, value }),
            });
        } catch (err) {
            console.error("Failed to save preference:", err);
        }
    }, []);

    return { getPreference, setPreference };
}

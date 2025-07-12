import { useEffect, useState } from "react";
import { fetchStatistics, Statistic } from "../api/statistics";

export function useStatistics() {
    const [statistics, setStatistics] = useState<Statistic[]>([]);

    useEffect(() => {
        async function loadStatistics() {
            try {
                const data = await fetchStatistics();
                console.log("Data fetched from API:", data);
                setStatistics(data);
            } catch (error) {
                console.error(error);
            }
        }

        loadStatistics();
    }, []);

    return {
        statistics: statistics,
    };
}

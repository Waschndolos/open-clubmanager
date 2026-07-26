import { useEffect, useState } from "react";
import { fetchStatistics, fetchChartStatistics, Statistic, DashboardCharts } from "../api/statistics";

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

export function useChartStatistics() {
    const [charts, setCharts] = useState<DashboardCharts | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        async function loadCharts() {
            try {
                const data = await fetchChartStatistics();
                setCharts(data);
            } catch (error) {
                console.error("Failed to load chart statistics:", error);
            } finally {
                setLoading(false);
            }
        }

        loadCharts();
    }, []);

    return { charts, loading };
}

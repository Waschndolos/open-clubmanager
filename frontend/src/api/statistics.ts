import {BACKEND_URL} from "./api";

export interface Statistic {
    id: number;
    value: number;
    details?: string[];
}

const BASE_URL = `${BACKEND_URL}/statistics`;
export async function fetchStatistics(): Promise<Statistic[]> {
    const response = await fetch(BASE_URL);
    if (!response.ok) {
        throw new Error('Error fetching statistics: ' + response.statusText);
    }
    return response.json();
}
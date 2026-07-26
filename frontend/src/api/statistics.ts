import {BACKEND_URL} from "./api";
import api from "./api";

export interface Statistic {
    id: number;
    value: number;
    details?: string[];
}

export interface MemberGrowthPoint {
    month: string;
    entries: number;
    exits: number;
}

export interface FinanceTimeSeriesPoint {
    month: string;
    income: number;
    expenses: number;
}

export interface FeeStatus {
    paid: number;
    open: number;
}

export interface SectionCount {
    section: string;
    count: number;
}

export interface DashboardCharts {
    memberGrowth: MemberGrowthPoint[];
    financeTimeSeries: FinanceTimeSeriesPoint[];
    feeStatus: FeeStatus;
    membersBySection: SectionCount[];
}

const BASE_URL = `${BACKEND_URL}/statistics`;

export async function fetchStatistics(): Promise<Statistic[]> {
    const response = await fetch(BASE_URL);
    if (!response.ok) {
        throw new Error('Error fetching statistics: ' + response.statusText);
    }
    return response.json();
}

export async function fetchChartStatistics(): Promise<DashboardCharts> {
    const response = await api.get<DashboardCharts>('/statistics/charts');
    return response.data;
}

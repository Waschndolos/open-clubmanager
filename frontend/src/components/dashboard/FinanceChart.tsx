import React from 'react';
import {
    BarChart,
    Bar,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    Legend,
    ResponsiveContainer,
} from 'recharts';
import { Box, Paper, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { FinanceTimeSeriesPoint } from '../../api/statistics';

interface FinanceChartProps {
    data: FinanceTimeSeriesPoint[];
}

export default function FinanceChart({ data }: FinanceChartProps) {
    const { t } = useTranslation();

    return (
        <Paper elevation={0} sx={{ p: 3, borderRadius: 3, border: '1px solid', borderColor: 'divider' }}>
            <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 2, textTransform: 'uppercase', letterSpacing: '0.05em', fontSize: '0.7rem', color: 'text.secondary' }}>
                {t('dashboard.charts.finance.title')}
            </Typography>
            <Box sx={{ width: '100%', height: 220 }}>
                <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={data} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                        <CartesianGrid strokeDasharray="3 3" stroke="#e0e0e0" />
                        <XAxis dataKey="month" tick={{ fontSize: 11 }} />
                        <YAxis tickFormatter={(v) => `${v} €`} tick={{ fontSize: 11 }} />
                        <Tooltip formatter={(value) => typeof value === 'number' ? `${value.toFixed(2)} €` : `${value} €`} />
                        <Legend wrapperStyle={{ fontSize: 12 }} />
                        <Bar
                            dataKey="income"
                            name={t('dashboard.charts.finance.income')}
                            fill="#4caf50"
                        />
                        <Bar
                            dataKey="expenses"
                            name={t('dashboard.charts.finance.expenses')}
                            fill="#f44336"
                        />
                    </BarChart>
                </ResponsiveContainer>
            </Box>
        </Paper>
    );
}

import React from 'react';
import {
    LineChart,
    Line,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    Legend,
    ResponsiveContainer,
} from 'recharts';
import { Box, Paper, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { MemberGrowthPoint } from '../../api/statistics';

interface MemberGrowthChartProps {
    data: MemberGrowthPoint[];
}

export default function MemberGrowthChart({ data }: MemberGrowthChartProps) {
    const { t } = useTranslation();

    return (
        <Paper elevation={0} sx={{ p: 3, borderRadius: 3, border: '1px solid', borderColor: 'divider' }}>
            <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 2, textTransform: 'uppercase', letterSpacing: '0.05em', fontSize: '0.7rem', color: 'text.secondary' }}>
                {t('dashboard.charts.memberGrowth.title')}
            </Typography>
            <Box sx={{ width: '100%', height: 220 }}>
                <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={data} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                        <CartesianGrid strokeDasharray="3 3" stroke="#e0e0e0" />
                        <XAxis dataKey="month" tick={{ fontSize: 11 }} />
                        <YAxis allowDecimals={false} tick={{ fontSize: 11 }} />
                        <Tooltip />
                        <Legend wrapperStyle={{ fontSize: 12 }} />
                        <Line
                            type="monotone"
                            dataKey="entries"
                            name={t('dashboard.charts.memberGrowth.entries')}
                            stroke="#4caf50"
                            strokeWidth={2}
                            dot={false}
                        />
                        <Line
                            type="monotone"
                            dataKey="exits"
                            name={t('dashboard.charts.memberGrowth.exits')}
                            stroke="#f44336"
                            strokeWidth={2}
                            dot={false}
                        />
                    </LineChart>
                </ResponsiveContainer>
            </Box>
        </Paper>
    );
}

import React from 'react';
import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Box, Paper, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { SectionCount } from '../../api/statistics';

interface MembersBySectionChartProps {
    data: SectionCount[];
}

const COLORS = ['#2196f3', '#9c27b0', '#ff9800', '#00bcd4', '#e91e63', '#4caf50', '#795548', '#607d8b'];

export default function MembersBySectionChart({ data }: MembersBySectionChartProps) {
    const { t } = useTranslation();

    const chartData = data.map((d) => ({ name: d.section, value: d.count }));

    return (
        <Paper elevation={0} sx={{ p: 3, borderRadius: 3, border: '1px solid', borderColor: 'divider' }}>
            <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 2, textTransform: 'uppercase', letterSpacing: '0.05em', fontSize: '0.7rem', color: 'text.secondary' }}>
                {t('dashboard.charts.membersBySection.title')}
            </Typography>
            <Box sx={{ width: '100%', height: 220 }}>
                <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                        <Pie
                            data={chartData}
                            cx="50%"
                            cy="50%"
                            innerRadius={55}
                            outerRadius={80}
                            paddingAngle={3}
                            dataKey="value"
                        >
                            {chartData.map((_, index) => (
                                <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                            ))}
                        </Pie>
                        <Tooltip formatter={(value: number) => value} />
                        <Legend wrapperStyle={{ fontSize: 12 }} />
                    </PieChart>
                </ResponsiveContainer>
            </Box>
        </Paper>
    );
}

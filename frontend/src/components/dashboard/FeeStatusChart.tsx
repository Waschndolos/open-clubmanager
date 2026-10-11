import React from 'react';
import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { useTranslation } from 'react-i18next';
import { FeeStatus } from '../../api/statistics';
import ChartCard, { useChartStyle } from './ChartCard';

interface FeeStatusChartProps {
    data: FeeStatus;
}

export default function FeeStatusChart({ data }: FeeStatusChartProps) {
    const { t } = useTranslation();
    const s = useChartStyle();

    const chartData = [
        { name: t('dashboard.charts.feeStatus.paid'), value: data.paid, color: s.positive },
        { name: t('dashboard.charts.feeStatus.open'), value: data.open, color: s.accent },
    ];

    return (
        <ChartCard title={t('dashboard.charts.feeStatus.title')} empty={data.paid + data.open === 0}>
            <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                    <Pie data={chartData} innerRadius="58%" outerRadius="82%" paddingAngle={data.paid && data.open ? 2 : 0} dataKey="value" stroke="none">
                        {chartData.map((d) => <Cell key={d.name} fill={d.color} />)}
                    </Pie>
                    <Tooltip {...s.tooltip} />
                    <Legend wrapperStyle={s.legend} iconType="circle" iconSize={8} />
                </PieChart>
            </ResponsiveContainer>
        </ChartCard>
    );
}

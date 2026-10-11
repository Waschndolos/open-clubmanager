import React from 'react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { useTranslation } from 'react-i18next';
import { MemberGrowthPoint } from '../../api/statistics';
import ChartCard, { useChartStyle } from './ChartCard';

interface MemberGrowthChartProps {
    data: MemberGrowthPoint[];
}

export default function MemberGrowthChart({ data }: MemberGrowthChartProps) {
    const { t } = useTranslation();
    const s = useChartStyle();

    return (
        <ChartCard title={t('dashboard.charts.memberGrowth.title')} empty={data.length === 0}>
            <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={data} margin={{ top: 5, right: 8, left: -16, bottom: 0 }}>
                    <defs>
                        <linearGradient id="mg-entries" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="0%" stopColor={s.positive} stopOpacity={0.25} />
                            <stop offset="100%" stopColor={s.positive} stopOpacity={0} />
                        </linearGradient>
                    </defs>
                    <CartesianGrid stroke={s.grid} vertical={false} />
                    <XAxis dataKey="month" tick={s.tick} axisLine={false} tickLine={false} />
                    <YAxis allowDecimals={false} tick={s.tick} axisLine={false} tickLine={false} />
                    <Tooltip {...s.tooltip} />
                    <Legend wrapperStyle={s.legend} iconType="circle" iconSize={8} />
                    <Area
                        type="monotone"
                        dataKey="entries"
                        name={t('dashboard.charts.memberGrowth.entries')}
                        stroke={s.positive}
                        fill="url(#mg-entries)"
                        strokeWidth={2}
                    />
                    <Area
                        type="monotone"
                        dataKey="exits"
                        name={t('dashboard.charts.memberGrowth.exits')}
                        stroke={s.negative}
                        fill="transparent"
                        strokeWidth={2}
                        strokeDasharray="4 3"
                    />
                </AreaChart>
            </ResponsiveContainer>
        </ChartCard>
    );
}

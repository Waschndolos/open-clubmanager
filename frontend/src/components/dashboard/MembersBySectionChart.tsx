import React from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { useTranslation } from 'react-i18next';
import { SectionCount } from '../../api/statistics';
import ChartCard, { useChartStyle } from './ChartCard';

interface MembersBySectionChartProps {
    data: SectionCount[];
}

/** Horizontal bars read better than a pie once a club has more than a few sections. */
export default function MembersBySectionChart({ data }: MembersBySectionChartProps) {
    const { t } = useTranslation();
    const s = useChartStyle();
    const chartData = [...data].sort((a, b) => b.count - a.count);
    const empty = chartData.every((d) => d.count === 0);

    return (
        <ChartCard title={t('dashboard.charts.membersBySection.title')} empty={empty}>
            <ResponsiveContainer width="100%" height="100%">
                <BarChart data={chartData} layout="vertical" margin={{ top: 0, right: 16, left: 0, bottom: 0 }}>
                    <XAxis type="number" allowDecimals={false} hide />
                    <YAxis type="category" dataKey="section" tick={s.tick} axisLine={false} tickLine={false} width={96} />
                    <Tooltip {...s.tooltip} cursor={{ fill: s.grid, opacity: 0.4 }} />
                    <Bar dataKey="count" name={t('dashboard.stats.members')} radius={[0, 4, 4, 0]} maxBarSize={22} label={{ position: 'right', ...s.tick }}>
                        {chartData.map((d, i) => <Cell key={d.section} fill={s.series[i % s.series.length]} />)}
                    </Bar>
                </BarChart>
            </ResponsiveContainer>
        </ChartCard>
    );
}

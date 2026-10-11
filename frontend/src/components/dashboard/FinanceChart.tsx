import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { useTranslation } from 'react-i18next';
import { FinanceTimeSeriesPoint } from '../../api/statistics';
import ChartCard, { useChartStyle } from './ChartCard';

interface FinanceChartProps {
    data: FinanceTimeSeriesPoint[];
}

export default function FinanceChart({ data }: FinanceChartProps) {
    const { t, i18n } = useTranslation();
    const s = useChartStyle();
    const currency = new Intl.NumberFormat(i18n.language, { style: 'currency', currency: 'EUR' });
    const compact = new Intl.NumberFormat(i18n.language, { style: 'currency', currency: 'EUR', notation: 'compact', maximumFractionDigits: 0 });

    return (
        <ChartCard title={t('dashboard.charts.finance.title')} empty={data.length === 0}>
            <ResponsiveContainer width="100%" height="100%">
                <BarChart data={data} margin={{ top: 5, right: 8, left: 0, bottom: 0 }} barGap={4}>
                    <CartesianGrid stroke={s.grid} vertical={false} />
                    <XAxis dataKey="month" tick={s.tick} axisLine={false} tickLine={false} />
                    <YAxis tickFormatter={(v) => compact.format(v)} tick={s.tick} axisLine={false} tickLine={false} width={56} />
                    <Tooltip
                        {...s.tooltip}
                        cursor={{ fill: s.grid, opacity: 0.4 }}
                        formatter={(value) => typeof value === 'number' ? currency.format(value) : value}
                    />
                    <Legend wrapperStyle={s.legend} iconType="circle" iconSize={8} />
                    <Bar dataKey="income" name={t('dashboard.charts.finance.income')} fill={s.positive} radius={[4, 4, 0, 0]} maxBarSize={36} />
                    <Bar dataKey="expenses" name={t('dashboard.charts.finance.expenses')} fill={s.accent} radius={[4, 4, 0, 0]} maxBarSize={36} />
                </BarChart>
            </ResponsiveContainer>
        </ChartCard>
    );
}

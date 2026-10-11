import { Box, Paper, Typography } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import InsightsOutlinedIcon from '@mui/icons-material/InsightsOutlined';
import { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

type ChartCardProps = {
    title: string;
    /** Render an empty state instead of the chart. */
    empty?: boolean;
    children: ReactNode;
    height?: number;
};

/** Frame shared by the dashboard charts, with a quiet empty state. */
export default function ChartCard({ title, empty = false, children, height = 240 }: ChartCardProps) {
    const { t } = useTranslation();
    return (
        <Paper sx={{ p: { xs: 2, sm: 2.5 }, height: '100%', boxSizing: 'border-box' }}>
            <Typography variant="h6" component="h3" sx={{ fontSize: '1rem', mb: 2 }}>
                {title}
            </Typography>
            <Box sx={{ width: '100%', height }}>
                {empty ? (
                    <Box
                        sx={{
                            height: '100%',
                            display: 'flex',
                            flexDirection: 'column',
                            alignItems: 'center',
                            justifyContent: 'center',
                            gap: 1,
                            borderRadius: 2,
                            bgcolor: (theme) => theme.custom.sunken,
                            color: 'text.secondary',
                            textAlign: 'center',
                            px: 2,
                        }}
                    >
                        <InsightsOutlinedIcon sx={{ opacity: 0.6 }} />
                        <Typography variant="body2">{t('dashboard.charts.empty')}</Typography>
                    </Box>
                ) : children}
            </Box>
        </Paper>
    );
}

/** Colours and axis styling for recharts, taken from the active MUI theme. */
export function useChartStyle() {
    const theme = useTheme();
    const isDark = theme.palette.mode === 'dark';
    return {
        positive: theme.palette.primary.main,
        negative: theme.palette.error.main,
        accent: theme.custom.accent,
        series: isDark
            ? ['#5BC49A', '#E3A72F', '#6AB7E8', '#C99BE0', '#F08A7E', '#9FB7AC']
            : ['#1E5A46', '#C98A12', '#2B6CB0', '#8E5BA8', '#C0392B', '#7C8C85'],
        grid: theme.palette.divider,
        tick: { fontSize: 12, fill: theme.palette.text.secondary },
        tooltip: {
            contentStyle: {
                background: theme.palette.background.paper,
                border: `1px solid ${theme.palette.divider}`,
                borderRadius: 10,
                fontSize: 13,
                color: theme.palette.text.primary,
            },
            labelStyle: { color: theme.palette.text.secondary, marginBottom: 4 },
            itemStyle: { color: theme.palette.text.primary },
        },
        legend: { fontSize: 13, color: theme.palette.text.secondary },
    };
}

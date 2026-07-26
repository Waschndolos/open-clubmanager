import { Grid, Divider, Typography, Box } from '@mui/material';
import StatisticCard from '../../components/dashboard/StatisticsCard';
import MemberGrowthChart from '../../components/dashboard/MemberGrowthChart';
import FinanceChart from '../../components/dashboard/FinanceChart';
import FeeStatusChart from '../../components/dashboard/FeeStatusChart';
import MembersBySectionChart from '../../components/dashboard/MembersBySectionChart';
import { useStatistics, useChartStatistics } from '../../hooks/statistics';
import PageHeader from '../../components/common/PageHeader';
import GridViewIcon from '@mui/icons-material/GridView';
import { useTranslation } from 'react-i18next';

export default function Dashboard() {
    const { statistics } = useStatistics();
    const { charts } = useChartStatistics();
    const { t } = useTranslation();

    return (
        <>
            <PageHeader
                title={t('menu.dashboard')}
                icon={<GridViewIcon fontSize="small" />}
            />
            <Grid container spacing={3}>
                {statistics.map(stat => (
                    <Grid size={{xs:12, md: 4}} key={stat.id}>
                        <StatisticCard
                            id={stat.id}
                            value={stat.value}
                            details={stat.details}
                        />
                    </Grid>
                ))}
            </Grid>

            {charts && (
                <Box sx={{ mt: 4 }}>
                    <Divider sx={{ mb: 3 }} />
                    <Typography variant="h6" fontWeight={700} sx={{ mb: 3 }}>
                        {t('dashboard.charts.title')}
                    </Typography>
                    <Grid container spacing={3}>
                        <Grid size={{ xs: 12, lg: 8 }}>
                            <MemberGrowthChart data={charts.memberGrowth} />
                        </Grid>
                        <Grid size={{ xs: 12, lg: 4 }}>
                            <FeeStatusChart data={charts.feeStatus} />
                        </Grid>
                        <Grid size={{ xs: 12, lg: 8 }}>
                            <FinanceChart data={charts.financeTimeSeries} />
                        </Grid>
                        <Grid size={{ xs: 12, lg: 4 }}>
                            <MembersBySectionChart data={charts.membersBySection} />
                        </Grid>
                    </Grid>
                </Box>
            )}
        </>
    );
}

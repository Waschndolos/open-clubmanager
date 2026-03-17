import {Grid} from '@mui/material';
import StatisticCard from '../../components/dashboard/StatisticsCard';
import {useStatistics} from '../../hooks/statistics';
import PageHeader from '../../components/common/PageHeader';
import GridViewIcon from '@mui/icons-material/GridView';
import {useTranslation} from 'react-i18next';

export default function Dashboard() {
    const { statistics } = useStatistics();
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
        </>
    );
}

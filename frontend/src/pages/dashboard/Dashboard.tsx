import {Grid} from '@mui/material';
import StatisticCard from '../../components/dashboard/StatisticsCard';
import {useStatistics} from '../../hooks/statistics';

export default function Dashboard() {
    const { statistics } = useStatistics();

    return (
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
    );
}

import {Box, Divider, Paper, Typography} from '@mui/material';
import GroupsIcon from '@mui/icons-material/Groups';
import AttachMoneyIcon from '@mui/icons-material/AttachMoney';
import SettingsIcon from '@mui/icons-material/Settings';
import {useTranslation} from "react-i18next";
import Grid from '@mui/material/Grid'


export default function Dashboard() {

    const {t} = useTranslation();

    return (
        <Box sx={{p: 4}}>
            <Typography variant="h4" gutterBottom>
                {t('dashboard.welcome')}
            </Typography>

            <Typography variant="body1" sx={{mb: 4}}>
                {t('dashboard.description')}
            </Typography>

            <Divider sx={{mb: 4}}/>

            <Grid container spacing={3}>
                <Grid size={{xs: 12, md: 4}}>
                    <Paper sx={{
                        p: 3,
                        backgroundColor: (theme) => theme.palette.mode === 'dark' ? '#23243a' : '#fff',
                        borderRadius: 4,
                        boxShadow: (theme) => theme.palette.mode === 'dark' ? '0 4px 24px 0 rgba(142,202,230,0.10)' : '0 2px 8px 0 rgba(0,0,0,0.10)',
                        border: (theme) => theme.palette.mode === 'dark' ? '1px solid #31324b' : '1px solid #e0e7ef',
                        transition: 'background 0.3s',
                    }}>
                        <GroupsIcon fontSize="large" color="primary"/>
                        <Typography variant="h6" sx={{mt: 2}}>
                            {t('dashboard.members.title')}
                        </Typography>
                        <Typography variant="body2" sx={{mt: 1}}>
                            {t('dashboard.members.description')}
                        </Typography>
                    </Paper>
                </Grid>

                <Grid size={{xs: 12, md: 4}}>
                    <Paper sx={{
                        p: 3,
                        backgroundColor: (theme) => theme.palette.mode === 'dark' ? '#23243a' : '#fff',
                        borderRadius: 4,
                        boxShadow: (theme) => theme.palette.mode === 'dark' ? '0 4px 24px 0 rgba(142,202,230,0.10)' : '0 2px 8px 0 rgba(0,0,0,0.10)',
                        border: (theme) => theme.palette.mode === 'dark' ? '1px solid #31324b' : '1px solid #e0e7ef',
                        transition: 'background 0.3s',
                    }}>
                        <AttachMoneyIcon fontSize="large" color="success"/>
                        <Typography variant="h6" sx={{mt: 2}}>
                            {t('dashboard.finance.title')}
                        </Typography>
                        <Typography variant="body2" sx={{mt: 1}}>
                            {t('dashboard.finance.description')}
                        </Typography>
                    </Paper>
                </Grid>

                <Grid size={{xs: 12, md: 4}}>
                    <Paper sx={{
                        p: 3,
                        backgroundColor: (theme) => theme.palette.mode === 'dark' ? '#23243a' : '#fff',
                        borderRadius: 4,
                        boxShadow: (theme) => theme.palette.mode === 'dark' ? '0 4px 24px 0 rgba(142,202,230,0.10)' : '0 2px 8px 0 rgba(0,0,0,0.10)',
                        border: (theme) => theme.palette.mode === 'dark' ? '1px solid #31324b' : '1px solid #e0e7ef',
                        transition: 'background 0.3s',
                    }}>
                        <SettingsIcon fontSize="large" color="success"/>
                        <Typography variant="h6" sx={{mt: 2}}>
                            {t('dashboard.settings.title')}
                        </Typography>
                        <Typography variant="body2" sx={{mt: 1}}>
                            {t('dashboard.settings.description')}
                        </Typography>
                    </Paper>
                </Grid>
            </Grid>
        </Box>
    );
}

import { useEffect, useState } from 'react';
import { Box, ButtonBase, Paper, Typography } from '@mui/material';
import WarningAmberOutlined from '@mui/icons-material/WarningAmberOutlined';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { fetchHouseholds, Household, memberName } from '../../api/households';

/** Dashboard reminder for households whose main payer left; hidden when none or not permitted. */
export default function HouseholdAlerts() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const [list, setList] = useState<Household[]>([]);

    useEffect(() => {
        fetchHouseholds()
            .then((all) => setList(all.filter((h) => h.payerMissing)))
            .catch(() => setList([]));
    }, []);

    if (list.length === 0) return null;

    return (
        <Box component="section" sx={{ mt: { xs: 4, sm: 5 } }}>
            <Typography variant="h6" component="h2" sx={{ mb: 1.5 }}>{t('households.dashboard.title')}</Typography>
            <Paper variant="outlined" sx={{ borderRadius: 3.5, p: 1 }}>
                {list.map((h) => (
                    <Box key={h.id} sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 1.5, py: 1 }}>
                        <WarningAmberOutlined color="warning" fontSize="small" />
                        <Typography sx={{ flexGrow: 1, fontWeight: 600 }}>{h.name}</Typography>
                        <Typography color="text.secondary" sx={{ fontSize: '0.9rem' }}>
                            {h.members.filter((m) => m.active).map(memberName).join(', ')}
                        </Typography>
                    </Box>
                ))}
                <ButtonBase onClick={() => navigate('/households')} sx={{ px: 1.5, py: 1, color: 'primary.main', fontWeight: 600 }}>
                    {t('households.dashboard.action')}
                </ButtonBase>
            </Paper>
        </Box>
    );
}

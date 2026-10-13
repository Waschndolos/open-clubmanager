import { useEffect, useState } from 'react';
import { Box, ButtonBase, Paper, Typography } from '@mui/material';
import BuildOutlined from '@mui/icons-material/BuildOutlined';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { fetchMaintenanceStates, MaintenanceState } from '../../api/maintenance';

const SHOWN = 5;

/** Dashboard reminder for overdue and soon due inspections; hidden when none or not permitted. */
export default function MaintenanceAlerts() {
    const { t, i18n } = useTranslation();
    const navigate = useNavigate();
    const [list, setList] = useState<MaintenanceState[]>([]);

    useEffect(() => {
        fetchMaintenanceStates()
            .then((all) => setList(all.filter((s) => s.due === 'due' || s.due === 'overdue')))
            .catch(() => setList([]));
    }, []);

    if (list.length === 0) return null;

    return (
        <Box component="section" sx={{ mt: { xs: 4, sm: 5 } }}>
            <Typography variant="h6" component="h2" sx={{ mb: 1.5 }}>{t('inventory.maintenance.dashboardTitle')}</Typography>
            <Paper variant="outlined" sx={{ borderRadius: 3.5, p: 1 }}>
                {list.slice(0, SHOWN).map((s) => (
                    <Box key={s.itemId} sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 1.5, py: 1 }}>
                        <BuildOutlined color={s.due === 'overdue' ? 'error' : 'warning'} fontSize="small" />
                        <Typography sx={{ flexGrow: 1, fontWeight: 600 }}>{s.name}</Typography>
                        <Typography color="text.secondary" sx={{ fontSize: '0.9rem' }}>
                            {t(`inventory.due.${s.due}`)}
                            {s.nextDue && ` · ${new Date(`${s.nextDue}T00:00:00`).toLocaleDateString(i18n.language)}`}
                        </Typography>
                    </Box>
                ))}
                <ButtonBase onClick={() => navigate('/inventory')} sx={{ px: 1.5, py: 1, color: 'primary.main', fontWeight: 600 }}>
                    {t('inventory.maintenance.dashboardAll', { count: list.length })}
                </ButtonBase>
            </Paper>
        </Box>
    );
}

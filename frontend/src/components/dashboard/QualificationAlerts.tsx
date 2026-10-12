import { useEffect, useState } from 'react';
import { Box, ButtonBase, Paper, Typography } from '@mui/material';
import WarningAmberOutlined from '@mui/icons-material/WarningAmberOutlined';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { fetchQualifications, Qualification } from '../../api/qualifications';

const SHOWN = 5;

/** Dashboard reminder for expired and soon expiring qualifications; hidden when none or not permitted. */
export default function QualificationAlerts() {
    const { t, i18n } = useTranslation();
    const navigate = useNavigate();
    const [list, setList] = useState<Qualification[]>([]);

    useEffect(() => {
        fetchQualifications()
            .then((all) => setList(all.filter((q) => q.status !== 'valid')))
            .catch(() => setList([]));
    }, []);

    if (list.length === 0) return null;

    return (
        <Box component="section" sx={{ mt: { xs: 4, sm: 5 } }}>
            <Typography variant="h6" component="h2" sx={{ mb: 1.5 }}>{t('qualifications.dashboard.title')}</Typography>
            <Paper variant="outlined" sx={{ borderRadius: 3.5, p: 1 }}>
                {list.slice(0, SHOWN).map((q) => (
                    <Box key={q.id} sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 1.5, py: 1 }}>
                        <WarningAmberOutlined color={q.status === 'expired' ? 'error' : 'warning'} fontSize="small" />
                        <Typography sx={{ flexGrow: 1, fontWeight: 600 }}>
                            {q.member.firstName} {q.member.lastName} – {q.type.name}
                        </Typography>
                        <Typography color="text.secondary" sx={{ fontSize: '0.9rem' }}>
                            {t(`qualifications.status.${q.status}`)}
                            {q.expiryDate && ` · ${new Date(`${q.expiryDate}T00:00:00`).toLocaleDateString(i18n.language)}`}
                        </Typography>
                    </Box>
                ))}
                <ButtonBase onClick={() => navigate('/qualifications')} sx={{ px: 1.5, py: 1, color: 'primary.main', fontWeight: 600 }}>
                    {t('qualifications.dashboard.all', { count: list.length })}
                </ButtonBase>
            </Paper>
        </Box>
    );
}

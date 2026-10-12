import { useEffect, useState } from 'react';
import { Box, ButtonBase, Paper, Typography } from '@mui/material';
import CelebrationOutlined from '@mui/icons-material/CelebrationOutlined';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { DEFAULT_AGES, DEFAULT_YEARS, fetchJubilees, isoDate, Jubilee } from '../../api/honors';

const SHOWN = 5;
const DAYS = 60;

/** Dashboard tile with the next jubilees; hidden when there are none or the user may not see them. */
export default function UpcomingJubilees() {
    const { t, i18n } = useTranslation();
    const navigate = useNavigate();
    const [list, setList] = useState<Jubilee[]>([]);

    useEffect(() => {
        const from = new Date();
        const to = new Date();
        to.setDate(to.getDate() + DAYS);
        fetchJubilees({ from: isoDate(from), to: isoDate(to), years: DEFAULT_YEARS, ages: DEFAULT_AGES })
            .then(setList)
            .catch(() => setList([]));
    }, []);

    if (list.length === 0) return null;

    return (
        <Box component="section" sx={{ mt: { xs: 4, sm: 5 } }}>
            <Typography variant="h6" component="h2" sx={{ mb: 1.5 }}>{t('honors.dashboard.title')}</Typography>
            <Paper variant="outlined" sx={{ borderRadius: 3.5, p: 1 }}>
                {list.slice(0, SHOWN).map((j) => (
                    <Box key={`${j.member.id}-${j.kind}-${j.milestone}`} sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 1.5, py: 1 }}>
                        <CelebrationOutlined color="primary" fontSize="small" />
                        <Typography sx={{ flexGrow: 1, fontWeight: 600 }}>
                            {j.member.firstName} {j.member.lastName}
                        </Typography>
                        <Typography color="text.secondary" sx={{ fontSize: '0.9rem' }}>
                            {t(`honors.kind.${j.kind}`, { count: j.milestone })}
                            {' · '}
                            {new Date(`${j.date}T00:00:00`).toLocaleDateString(i18n.language)}
                        </Typography>
                    </Box>
                ))}
                <ButtonBase onClick={() => navigate('/honors')} sx={{ px: 1.5, py: 1, color: 'primary.main', fontWeight: 600 }}>
                    {t('honors.dashboard.all', { count: list.length })}
                </ButtonBase>
            </Paper>
        </Box>
    );
}

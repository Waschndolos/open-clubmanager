import { useEffect, useState } from 'react';
import { Box, ButtonBase, Paper, Typography } from '@mui/material';
import EventOutlined from '@mui/icons-material/EventOutlined';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { addDays, EventOccurrence, fetchOccurrences, ymd } from '../../api/events';

const SHOWN = 5;
const DAYS = 30;

/** Dashboard tile with the next events; hidden when none or not permitted. */
export default function UpcomingEvents() {
    const { t, i18n } = useTranslation();
    const navigate = useNavigate();
    const [list, setList] = useState<EventOccurrence[]>([]);

    useEffect(() => {
        const now = new Date();
        fetchOccurrences(ymd(now), ymd(addDays(now, DAYS)))
            .then((all) => setList(all.filter((o) => new Date(o.end) > now).slice(0, SHOWN)))
            .catch(() => setList([]));
    }, []);

    if (list.length === 0) return null;

    return (
        <Box component="section" sx={{ mt: { xs: 4, sm: 5 } }}>
            <Typography variant="h6" component="h2" sx={{ mb: 1.5 }}>{t('events.dashboard.title')}</Typography>
            <Paper variant="outlined" sx={{ borderRadius: 3.5, p: 1 }}>
                {list.map((o) => {
                    const start = new Date(o.start);
                    return (
                        <Box key={`${o.eventId}-${o.occurrenceDate}`} sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 1.5, py: 1 }}>
                            <EventOutlined color="primary" fontSize="small" />
                            <Typography sx={{ flexGrow: 1, fontWeight: 600 }} noWrap>{o.title}</Typography>
                            <Typography color="text.secondary" sx={{ fontSize: '0.9rem', whiteSpace: 'nowrap' }}>
                                {start.toLocaleDateString(i18n.language, { weekday: 'short', day: 'numeric', month: 'short' })}
                                {!o.allDay && ` · ${start.toLocaleTimeString(i18n.language, { hour: '2-digit', minute: '2-digit' })}`}
                            </Typography>
                        </Box>
                    );
                })}
                <ButtonBase onClick={() => navigate('/events')} sx={{ px: 1.5, py: 1, color: 'primary.main', fontWeight: 600 }}>
                    {t('events.dashboard.all')}
                </ButtonBase>
            </Paper>
        </Box>
    );
}

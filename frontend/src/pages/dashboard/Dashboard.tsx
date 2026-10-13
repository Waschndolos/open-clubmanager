import { Box, Button, ButtonBase, Grid, LinearProgress, Paper, Skeleton, Typography } from '@mui/material';
import { alpha, useTheme } from '@mui/material/styles';
import AccountBalanceOutlined from '@mui/icons-material/AccountBalanceOutlined';
import AccountTreeOutlined from '@mui/icons-material/AccountTreeOutlined';
import ChevronRightRounded from '@mui/icons-material/ChevronRightRounded';
import GroupsOutlined from '@mui/icons-material/GroupsOutlined';
import HistoryOutlined from '@mui/icons-material/HistoryOutlined';
import PersonAddAlt1Outlined from '@mui/icons-material/PersonAddAlt1Outlined';
import { ReactElement } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import MemberGrowthChart from '../../components/dashboard/MemberGrowthChart';
import FinanceChart from '../../components/dashboard/FinanceChart';
import FeeStatusChart from '../../components/dashboard/FeeStatusChart';
import MyTasks from '../../components/dashboard/MyTasks';
import UpcomingEvents from '../../components/dashboard/UpcomingEvents';
import UpcomingJubilees from '../../components/dashboard/UpcomingJubilees';
import QualificationAlerts from '../../components/dashboard/QualificationAlerts';
import MaintenanceAlerts from '../../components/dashboard/MaintenanceAlerts';
import HouseholdAlerts from '../../components/dashboard/HouseholdAlerts';
import MembersBySectionChart from '../../components/dashboard/MembersBySectionChart';
import { useNavGroups } from '../../components/menu/navItems';
import { useStatistics, useChartStatistics } from '../../hooks/statistics';

// Statistic ids as delivered by GET /statistics (server/internal/handlers/statistics.go).
const STAT = { members: 1, activeMembers: 2, groups: 3, roles: 4, sections: 5, fees: 6, paidFees: 7 } as const;

function greetingKey(hour: number) {
    if (hour < 11) return 'dashboard.greeting.morning';
    if (hour < 18) return 'dashboard.greeting.afternoon';
    return 'dashboard.greeting.evening';
}

export default function Dashboard() {
    const { statistics } = useStatistics();
    const { charts } = useChartStatistics();
    const { t, i18n } = useTranslation();
    const theme = useTheme();
    const navigate = useNavigate();
    const navLinks = useNavGroups().flatMap((g) => g.items.map((i) => i.link));

    const loaded = statistics.length > 0;
    const stat = (id: number) => statistics.find((s) => s.id === id)?.value ?? 0;
    const members = stat(STAT.members);
    const active = stat(STAT.activeMembers);
    const fees = stat(STAT.fees);
    const paidFees = stat(STAT.paidFees);
    const number = new Intl.NumberFormat(i18n.language);

    // The member panel is the one solid pine surface on the page.
    const hero = theme.palette.mode === 'dark'
        ? { bg: '#173F31', fg: '#E6EEE9' }
        : { bg: theme.palette.primary.main, fg: '#FFFFFF' };

    const now = new Date();
    const today = now.toLocaleDateString(i18n.language, { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });

    const quick: { label: string; icon: ReactElement; link: string }[] = [
        { label: t('dashboard.quick.members'), icon: <GroupsOutlined />, link: '/members' },
        { label: t('dashboard.quick.finance'), icon: <AccountBalanceOutlined />, link: '/finance' },
        { label: t('dashboard.quick.entities'), icon: <AccountTreeOutlined />, link: '/entities' },
        { label: t('dashboard.quick.history'), icon: <HistoryOutlined />, link: '/history' },
    ].filter((q) => navLinks.includes(q.link));

    return (
        <>
            <Box component="header" sx={{ mb: { xs: 3, sm: 4 } }}>
                <Typography color="text.secondary" sx={{ fontSize: '0.95rem', mb: 0.5 }}>
                    {today}
                </Typography>
                <Typography
                    variant="h1"
                    sx={{ fontSize: { xs: '2.25rem', sm: '3rem' }, lineHeight: 1.05 }}
                >
                    {t(greetingKey(now.getHours()))}
                </Typography>
            </Box>

            <Grid container spacing={2.5}>
                <Grid size={{ xs: 12, lg: 5 }}>
                    <Paper
                        sx={{
                            p: { xs: 2.5, sm: 3 },
                            height: '100%',
                            boxSizing: 'border-box',
                            display: 'flex',
                            flexDirection: 'column',
                            position: 'relative',
                            overflow: 'hidden',
                            bgcolor: hero.bg,
                            color: hero.fg,
                            border: 0,
                        }}
                    >
                        <Box
                            aria-hidden
                            component="svg"
                            viewBox="0 0 48 32"
                            sx={{ position: 'absolute', right: 24, top: 24, width: 44 }}
                        >
                            <rect x="0" y="0" width="2.5" height="32" rx="1.25" fill={hero.fg} opacity={0.5} />
                            <path d="M2.5 1.5 L44 10 L2.5 18.5 Z" fill={theme.custom.accent} />
                        </Box>
                        <Typography sx={{ fontWeight: 600, opacity: 0.85, mb: 1 }}>{t('dashboard.stats.members')}</Typography>
                        {loaded ? (
                            <Typography sx={{ fontFamily: theme.typography.h1.fontFamily, fontWeight: 700, fontSize: { xs: '3.5rem', sm: '4.5rem' }, lineHeight: 1, letterSpacing: '-0.04em' }}>
                                {number.format(members)}
                            </Typography>
                        ) : (
                            <Skeleton variant="text" width={120} sx={{ fontSize: '4.5rem', bgcolor: alpha('#fff', 0.15) }} />
                        )}

                        <Box sx={{ mt: 'auto', pt: 3 }}>
                            {loaded && members === 0 ? (
                                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2, flexWrap: 'wrap' }}>
                                    <Typography sx={{ opacity: 0.85 }}>{t('dashboard.stats.noMembers')}</Typography>
                                    <Button
                                        variant="contained"
                                        color="secondary"
                                        startIcon={<PersonAddAlt1Outlined />}
                                        onClick={() => navigate('/members')}
                                    >
                                        {t('dashboard.stats.addMembers')}
                                    </Button>
                                </Box>
                            ) : (
                                <>
                                    <Typography sx={{ fontWeight: 600, mb: 1 }}>
                                        {t('dashboard.stats.activeMembers', { count: active })}
                                    </Typography>
                                    <LinearProgress
                                        variant="determinate"
                                        value={members ? Math.min(100, (active / members) * 100) : 0}
                                        aria-label={t('dashboard.stats.activeShare')}
                                        sx={{
                                            height: 8,
                                            borderRadius: 4,
                                            bgcolor: alpha(hero.fg, 0.18),
                                            '& .MuiLinearProgress-bar': { borderRadius: 4, bgcolor: hero.fg },
                                        }}
                                    />
                                </>
                            )}
                        </Box>
                    </Paper>
                </Grid>

                <Grid size={{ xs: 12, lg: 7 }}>
                    <Grid container spacing={{ xs: 1.5, sm: 2.5 }} sx={{ height: '100%' }}>
                        <Grid size={4}>
                            <StatTile label={t('dashboard.stats.sections')} value={loaded ? number.format(stat(STAT.sections)) : null} onClick={() => navigate('/entities')} />
                        </Grid>
                        <Grid size={4}>
                            <StatTile label={t('dashboard.stats.groups')} value={loaded ? number.format(stat(STAT.groups)) : null} onClick={() => navigate('/entities')} />
                        </Grid>
                        <Grid size={4}>
                            <StatTile label={t('dashboard.stats.roles')} value={loaded ? number.format(stat(STAT.roles)) : null} onClick={() => navigate('/entities')} />
                        </Grid>
                        <Grid size={12}>
                            <Paper sx={{ p: { xs: 2, sm: 2.5 } }}>
                                <Box sx={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', gap: 2, mb: 1.5 }}>
                                    <Typography sx={{ fontWeight: 600, color: 'text.secondary' }}>{t('dashboard.stats.fees')}</Typography>
                                    {loaded ? (
                                        <Typography sx={{ fontFamily: theme.typography.h1.fontFamily, fontWeight: 700, fontSize: '1.75rem', lineHeight: 1 }}>
                                            {number.format(paidFees)}
                                            <Typography component="span" sx={{ ml: 1, fontSize: '0.95rem', fontWeight: 500, color: 'text.secondary', fontFamily: theme.typography.fontFamily }}>
                                                {t('dashboard.stats.feesOf', { total: number.format(fees) })}
                                            </Typography>
                                        </Typography>
                                    ) : <Skeleton width={80} />}
                                </Box>
                                <LinearProgress
                                    variant="determinate"
                                    value={fees ? Math.min(100, (paidFees / fees) * 100) : 0}
                                    aria-label={t('dashboard.stats.fees')}
                                    sx={{
                                        height: 8,
                                        borderRadius: 4,
                                        bgcolor: theme.custom.sunken,
                                        '& .MuiLinearProgress-bar': { borderRadius: 4 },
                                    }}
                                />
                            </Paper>
                        </Grid>
                    </Grid>
                </Grid>
            </Grid>

            {quick.length > 0 && (
                <Box component="section" sx={{ mt: { xs: 4, sm: 5 } }}>
                    <Typography variant="h6" component="h2" sx={{ mb: 1.5 }}>{t('dashboard.quick.title')}</Typography>
                    <Grid container spacing={1.5}>
                        {quick.map((q) => (
                            <Grid key={q.link} size={{ xs: 12, sm: 6, lg: 3 }}>
                                <ButtonBase
                                    onClick={() => navigate(q.link)}
                                    sx={{
                                        width: '100%',
                                        justifyContent: 'flex-start',
                                        gap: 1.5,
                                        p: 1.5,
                                        pr: 1,
                                        borderRadius: 3,
                                        border: 1,
                                        borderColor: 'divider',
                                        bgcolor: 'background.paper',
                                        textAlign: 'left',
                                        transition: 'border-color .15s, background-color .15s',
                                        '&:hover, &.Mui-focusVisible': { borderColor: alpha(theme.palette.primary.main, 0.5), bgcolor: alpha(theme.palette.primary.main, 0.04) },
                                        '&:hover .chev': { transform: 'translateX(2px)', color: 'primary.main' },
                                    }}
                                >
                                    <Box sx={{ display: 'flex', p: 1, borderRadius: 2, color: 'primary.main', bgcolor: alpha(theme.palette.primary.main, 0.1) }}>
                                        {q.icon}
                                    </Box>
                                    <Typography sx={{ fontWeight: 600, flexGrow: 1, fontSize: '0.95rem' }}>{q.label}</Typography>
                                    <ChevronRightRounded className="chev" sx={{ color: 'text.secondary', transition: 'transform .15s, color .15s' }} />
                                </ButtonBase>
                            </Grid>
                        ))}
                    </Grid>
                </Box>
            )}

            <UpcomingEvents />
            <MyTasks />
            <UpcomingJubilees />
            <QualificationAlerts />
            <MaintenanceAlerts />
            <HouseholdAlerts />

            {charts && (
                <Box component="section" sx={{ mt: { xs: 4, sm: 5 } }}>
                    <Typography variant="h6" component="h2" sx={{ mb: 1.5 }}>{t('dashboard.charts.title')}</Typography>
                    <Grid container spacing={2.5}>
                        <Grid size={{ xs: 12, lg: 8 }}>
                            <MemberGrowthChart data={charts.memberGrowth} />
                        </Grid>
                        <Grid size={{ xs: 12, md: 6, lg: 4 }}>
                            <FeeStatusChart data={charts.feeStatus} />
                        </Grid>
                        <Grid size={{ xs: 12, md: 6, lg: 8 }} sx={{ order: { md: 4, lg: 3 } }}>
                            <FinanceChart data={charts.financeTimeSeries} />
                        </Grid>
                        <Grid size={{ xs: 12, md: 6, lg: 4 }} sx={{ order: { md: 3, lg: 4 } }}>
                            <MembersBySectionChart data={charts.membersBySection} />
                        </Grid>
                    </Grid>
                </Box>
            )}
        </>
    );
}

function StatTile({ label, value, onClick }: { label: string; value: string | null; onClick: () => void }) {
    const theme = useTheme();
    return (
        <ButtonBase
            onClick={onClick}
            sx={{
                width: '100%',
                height: '100%',
                minHeight: { xs: 92, sm: 108 },
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'flex-start',
                justifyContent: 'space-between',
                p: { xs: 1.5, sm: 2.5 },
                borderRadius: 3.5,
                border: 1,
                borderColor: 'divider',
                bgcolor: 'background.paper',
                textAlign: 'left',
                transition: 'border-color .15s',
                '&:hover, &.Mui-focusVisible': { borderColor: alpha(theme.palette.primary.main, 0.5) },
            }}
        >
            <Typography noWrap sx={{ fontWeight: 600, color: 'text.secondary', fontSize: { xs: '0.85rem', sm: '1rem' }, maxWidth: '100%' }}>{label}</Typography>
            {value !== null ? (
                <Typography sx={{ fontFamily: theme.typography.h1.fontFamily, fontWeight: 700, fontSize: { xs: '1.75rem', sm: '2.25rem' }, lineHeight: 1, letterSpacing: '-0.03em' }}>
                    {value}
                </Typography>
            ) : <Skeleton width={48} sx={{ fontSize: '2.25rem' }} />}
        </ButtonBase>
    );
}

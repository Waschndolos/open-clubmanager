import { AppBar, Box, IconButton, Toolbar, Typography } from '@mui/material';
import MenuRoundedIcon from '@mui/icons-material/MenuRounded';
import { matchPath, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import NotificationBell from './NotificationBell';
import ThemeToggle from './ThemeToggle';
import SyncStatusButton from './SyncStatusButton';
import { useNavGroups } from '../menu/navItems';

type HeaderProps = {
    /** Shown on small screens where the sidebar is a drawer. */
    onOpenNavigation?: () => void;
};

/** Slim top bar: where you are on the left, status and quick toggles on the right. */
export default function Header({ onOpenNavigation }: HeaderProps) {
    const { t } = useTranslation();
    const { pathname } = useLocation();
    const groups = useNavGroups();

    let groupLabel: string | undefined;
    let pageLabel: string | undefined;
    for (const group of groups) {
        const item = group.items.find((i) => matchPath({ path: i.link, end: false }, pathname));
        if (item) {
            groupLabel = group.label;
            pageLabel = item.label;
            break;
        }
    }

    return (
        <AppBar position="sticky">
            <Toolbar sx={{ minHeight: { xs: 56, sm: 64 }, px: { xs: 1.5, sm: 3 }, gap: 1 }}>
                {onOpenNavigation && (
                    <IconButton edge="start" onClick={onOpenNavigation} aria-label={t('nav.openMenu')}>
                        <MenuRoundedIcon />
                    </IconButton>
                )}

                <Box
                    component="nav"
                    aria-label="Breadcrumb"
                    sx={{ flexGrow: 1, minWidth: 0, display: 'flex', alignItems: 'center', gap: 1 }}
                >
                    {groupLabel && (
                        <>
                            <Typography noWrap sx={{ fontSize: '0.9rem', color: 'text.secondary', display: { xs: 'none', sm: 'block' } }}>
                                {groupLabel}
                            </Typography>
                            <Typography aria-hidden sx={{ color: 'divider', display: { xs: 'none', sm: 'block' } }}>/</Typography>
                        </>
                    )}
                    {pageLabel && (
                        <Typography noWrap aria-current="page" sx={{ fontSize: '0.9rem', fontWeight: 600 }}>
                            {pageLabel}
                        </Typography>
                    )}
                </Box>

                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, color: 'text.secondary' }}>
                    <SyncStatusButton />
                    <NotificationBell />
                    <ThemeToggle />
                </Box>
            </Toolbar>
        </AppBar>
    );
}

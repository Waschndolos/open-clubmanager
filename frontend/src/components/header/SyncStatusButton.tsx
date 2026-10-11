import React, { useState } from 'react';
import { Alert, Box, Button, IconButton, Popover, Stack, Tooltip, Typography } from '@mui/material';
import CloudDoneOutlinedIcon from '@mui/icons-material/CloudDoneOutlined';
import CloudOffOutlinedIcon from '@mui/icons-material/CloudOffOutlined';
import CloudSyncOutlinedIcon from '@mui/icons-material/CloudSyncOutlined';
import ReportProblemOutlinedIcon from '@mui/icons-material/ReportProblemOutlined';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { CollaborationStatus } from '../../api/collaboration';
import { useCollaboration } from '../../context/CollaborationContext';

type State = 'ok' | 'syncing' | 'offline' | 'attention';

export function syncState(st: CollaborationStatus): State {
    if (st.syncing) return 'syncing';
    if (st.locked || st.mode === 'revoked' || st.conflicts > 0 || st.pendingRequests > 0 || st.lostChanges > 0 || st.badKeys > 0) {
        return 'attention';
    }
    if (st.storageError || st.lastError) return 'offline';
    return 'ok';
}

const icons: Record<State, React.ReactElement> = {
    ok: <CloudDoneOutlinedIcon fontSize="small" />,
    syncing: <CloudSyncOutlinedIcon fontSize="small" />,
    offline: <CloudOffOutlinedIcon fontSize="small" />,
    attention: <ReportProblemOutlinedIcon fontSize="small" />,
};

/** Header icon with the sync state and a popover with details and "Sync now" (UX F7). Hidden in local mode. */
export default function SyncStatusButton() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const { status, sync } = useCollaboration();
    const [anchor, setAnchor] = useState<HTMLElement | null>(null);

    if (!status || status.mode === 'local') return null;

    const state = syncState(status);
    const time = status.lastSync ? new Date(status.lastSync).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '';
    const tooltip =
        state === 'syncing' ? t('collaboration.status.syncing')
        : state === 'attention' ? t('collaboration.status.attention')
        : state === 'offline' ? t('collaboration.status.offline', { count: status.pendingChanges })
        : status.lastSync ? t('collaboration.status.upToDate', { time }) : t('collaboration.status.neverSynced');

    const go = (path: string) => {
        setAnchor(null);
        navigate(path);
    };

    return (
        <>
            <Tooltip title={tooltip}>
                <IconButton
                    color={state === 'attention' ? 'warning' : 'inherit'}
                    size="small"
                    aria-label={tooltip}
                    onClick={(e) => setAnchor(e.currentTarget)}
                    sx={state === 'syncing' ? {
                        '@keyframes ocmspin': { to: { transform: 'rotate(360deg)' } },
                        '& svg': { animation: 'ocmspin 1.2s linear infinite' },
                        '@media (prefers-reduced-motion: reduce)': { '& svg': { animation: 'none' } },
                    } : undefined}
                >
                    {icons[state]}
                </IconButton>
            </Tooltip>
            <Popover
                open={!!anchor}
                anchorEl={anchor}
                onClose={() => setAnchor(null)}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                transformOrigin={{ vertical: 'top', horizontal: 'right' }}
            >
                <Stack spacing={1.25} sx={{ p: 2, width: 340, maxWidth: '100vw' }}>
                    <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>{t('collaboration.title')}</Typography>
                    {status.locked && <Alert severity="warning">{t('collaboration.status.locked')}</Alert>}
                    {status.mode === 'revoked' && <Alert severity="error">{t('collaboration.status.revoked')}</Alert>}
                    {status.mode === 'joining' && (
                        <Alert severity="info" action={<Button size="small" onClick={() => go('/collaboration/join')}>{t('collaboration.popover.view')}</Button>}>
                            {t('collaboration.status.joining')}
                        </Alert>
                    )}
                    {!status.locked && (
                        <>
                            <Typography variant="body2">
                                {status.syncing ? t('collaboration.status.syncing')
                                    : t('collaboration.popover.lastSync', { time: status.lastSync ? new Date(status.lastSync).toLocaleString() : t('collaboration.settings.never') })}
                            </Typography>
                            {status.storagePath && (
                                <Typography variant="body2" color="text.secondary" sx={{ wordBreak: 'break-all' }}>
                                    {t('collaboration.popover.storage', { path: status.storagePath })}
                                </Typography>
                            )}
                            <Typography variant="body2" color="text.secondary">
                                {t('collaboration.popover.pending', { count: status.pendingChanges })}
                            </Typography>
                        </>
                    )}
                    {(status.storageError || status.lastError) && (
                        <Alert severity="warning">
                            {status.storageError ? t('collaboration.errors.STORAGE_MISSING') : t('collaboration.popover.error', { error: status.lastError })}
                        </Alert>
                    )}
                    {status.pendingRequests > 0 && (
                        <Alert severity="info" action={<Button size="small" onClick={() => go('/collaboration?tab=people')}>{t('collaboration.popover.view')}</Button>}>
                            {t('collaboration.popover.requests', { count: status.pendingRequests })}
                        </Alert>
                    )}
                    {status.conflicts > 0 && (
                        <Alert severity="warning" action={<Button size="small" onClick={() => go('/collaboration/conflicts')}>{t('collaboration.popover.view')}</Button>}>
                            {t('collaboration.popover.conflicts', { count: status.conflicts })}
                        </Alert>
                    )}
                    {status.lostChanges > 0 && <Alert severity="warning">{t('collaboration.popover.lostChanges')}</Alert>}
                    {status.badKeys > 0 && <Alert severity="error">{t('collaboration.popover.badKeys')}</Alert>}
                    <Box sx={{ display: 'flex', gap: 1, justifyContent: 'space-between', alignItems: 'center' }}>
                        <Button size="small" onClick={() => go('/collaboration')}>{t('collaboration.popover.settings')}</Button>
                        <Button variant="contained" size="small" onClick={sync} disabled={status.syncing || status.locked}>
                            {t('collaboration.popover.syncNow')}
                        </Button>
                    </Box>
                    <Typography variant="caption" color="text.secondary">{t('collaboration.popover.folderHint')}</Typography>
                </Stack>
            </Popover>
        </>
    );
}

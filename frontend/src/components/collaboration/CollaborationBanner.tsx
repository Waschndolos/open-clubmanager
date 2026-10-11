import React from 'react';
import { Alert, Button } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router-dom';
import { useCollaboration } from '../../context/CollaborationContext';

/** Shows above every page while this device cannot work normally in its club. */
export default function CollaborationBanner() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const { pathname } = useLocation();
    const { status } = useCollaboration();
    if (!status || status.mode === 'local' || pathname.startsWith('/collaboration')) return null;

    if (status.mode === 'joining') {
        return (
            <Alert severity="info" sx={{ mb: 2 }}
                   action={<Button color="inherit" size="small" onClick={() => navigate('/collaboration/join')}>{t('collaboration.popover.view')}</Button>}>
                {t('collaboration.status.joining')}
            </Alert>
        );
    }
    if (status.mode === 'revoked') return <Alert severity="error" sx={{ mb: 2 }}>{t('collaboration.status.revoked')}</Alert>;
    if (status.locked) return <Alert severity="warning" sx={{ mb: 2 }}>{t('collaboration.status.locked')}</Alert>;
    return null;
}

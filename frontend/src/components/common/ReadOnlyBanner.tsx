import React, { useCallback, useState } from 'react';
import { Alert, Button, CircularProgress } from '@mui/material';
import { LockOpen } from '@mui/icons-material';
import { useTranslation } from 'react-i18next';
import { StorageStatus } from '../../api/types';

interface ReadOnlyBannerProps {
    status: StorageStatus;
    onRequestEditMode: () => Promise<void>;
}

/**
 * Shown when the app is in read-only mode because another instance holds
 * the write lock.  Lets the user request edit mode when the lock is stale
 * or after the current holder releases it.
 */
const ReadOnlyBanner: React.FC<ReadOnlyBannerProps> = ({ status, onRequestEditMode }) => {
    const { t } = useTranslation();
    const [requesting, setRequesting] = useState(false);

    const holderLabel = status.lockHolder?.holderLabel ?? t('readOnlyBanner.unknownHolder');

    const handleRequest = useCallback(async () => {
        setRequesting(true);
        try {
            await onRequestEditMode();
        } finally {
            setRequesting(false);
        }
    }, [onRequestEditMode]);

    return (
        <Alert
            severity="warning"
            variant="filled"
            sx={{ borderRadius: 0, py: 0.5 }}
            action={
                <Button
                    color="inherit"
                    size="small"
                    startIcon={requesting ? <CircularProgress size={14} color="inherit" /> : <LockOpen />}
                    onClick={handleRequest}
                    disabled={requesting}
                >
                    {t('readOnlyBanner.requestEdit')}
                </Button>
            }
        >
            {t('readOnlyBanner.message', { holder: holderLabel })}
        </Alert>
    );
};

export default ReadOnlyBanner;

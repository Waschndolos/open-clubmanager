import React, { createContext, useCallback, useContext, useEffect, useState, ReactNode } from 'react';
import { StorageStatus } from '../api/types';
import { getDataClient } from '../api/clientFactory';
import { isElectronFolderMode } from '../lib/environment';

interface StorageContextType {
    status: StorageStatus | null;
    refreshStatus: () => Promise<void>;
    requestEditMode: () => Promise<void>;
    releaseEditMode: () => Promise<void>;
}

const StorageContext = createContext<StorageContextType | undefined>(undefined);

const POLL_INTERVAL_MS = 15_000;

export const StorageProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
    const [status, setStatus] = useState<StorageStatus | null>(null);

    const refreshStatus = useCallback(async () => {
        if (!isElectronFolderMode()) return;
        try {
            const s = await getDataClient().storage.getStatus();
            setStatus(s);
        } catch {
            // Ignore errors during background polling.
        }
    }, []);

    const requestEditMode = useCallback(async () => {
        const result = await getDataClient().storage.requestEditMode();
        setStatus(result.status);
    }, []);

    const releaseEditMode = useCallback(async () => {
        await getDataClient().storage.releaseEditMode();
        await refreshStatus();
    }, [refreshStatus]);

    useEffect(() => {
        if (!isElectronFolderMode()) return;

        refreshStatus();
        const timer = setInterval(refreshStatus, POLL_INTERVAL_MS);
        return () => clearInterval(timer);
    }, [refreshStatus]);

    return (
        <StorageContext.Provider value={{ status, refreshStatus, requestEditMode, releaseEditMode }}>
            {children}
        </StorageContext.Provider>
    );
};

export const useStorage = (): StorageContextType => {
    const context = useContext(StorageContext);
    if (!context) {
        throw new Error('useStorage must be used within StorageProvider');
    }
    return context;
};

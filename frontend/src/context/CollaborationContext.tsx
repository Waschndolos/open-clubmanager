import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { Area, AreaPermission, CollaborationStatus, getStatus, syncNow } from '../api/collaboration';
import { useAuth } from './AuthContext';

// Polls the collaboration status (UX F7) and exposes what the device may see
// and change (UX F10).

const POLL_MS = 30_000;

interface CollaborationContextType {
    status: CollaborationStatus | null;
    refresh: () => Promise<void>;
    sync: () => Promise<void>;
}

const CollaborationContext = createContext<CollaborationContextType>({
    status: null,
    refresh: async () => {},
    sync: async () => {},
});

export const CollaborationProvider = ({ children }: { children: React.ReactNode }) => {
    const { accessToken } = useAuth();
    const [status, setStatus] = useState<CollaborationStatus | null>(null);
    const active = useRef(true);

    const refresh = useCallback(async () => {
        try {
            const st = await getStatus();
            if (active.current) setStatus(st);
        } catch {
            // Keep the last known status; the next poll tries again.
        }
    }, []);

    const sync = useCallback(async () => {
        setStatus((s) => (s ? { ...s, syncing: true } : s));
        try {
            const st = await syncNow();
            if (active.current) setStatus(st);
        } catch {
            await refresh();
        }
    }, [refresh]);

    useEffect(() => {
        active.current = true;
        if (!accessToken) {
            setStatus(null);
            return;
        }
        refresh();
        const id = window.setInterval(refresh, POLL_MS);
        return () => {
            active.current = false;
            window.clearInterval(id);
        };
    }, [accessToken, refresh]);

    return (
        <CollaborationContext.Provider value={{ status, refresh, sync }}>
            {children}
        </CollaborationContext.Provider>
    );
};

export const useCollaboration = () => useContext(CollaborationContext);

/**
 * What this device may do in an area. In local mode, and while the status is
 * still loading, everything is writable, so the app behaves as before.
 */
export function usePermission(area: Area): AreaPermission {
    const { status } = useCollaboration();
    if (!status || status.mode === 'local') return 'write';
    return status.permissions[area];
}

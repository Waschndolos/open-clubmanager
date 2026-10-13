import React, { createContext, useCallback, useContext, useMemo, useState } from 'react';

export type Notification = {
    id: string;
    message: string;
};

type NotificationContextType = {
    notifications: Notification[];
    addNotification: (message: string) => void;
    clearNotifications: () => void;
};

const NotificationContext = createContext<NotificationContextType>({
    notifications: [],
    addNotification: () => {},
    clearNotifications: () => {},
});

export const NotificationProvider = ({ children }: { children: React.ReactNode }) => {
    const [notifications, setNotifications] = useState<Notification[]>([]);

    // Stable identities: components list addNotification in effect
    // dependencies, so a new function per render would re-run those effects
    // after every notification (and loop when the effect itself notifies).
    const addNotification = useCallback((message: string) => {
        setNotifications((prev) => [
            ...prev,
            { id: crypto.randomUUID(), message },
        ]);
    }, []);

    const clearNotifications = useCallback(() => setNotifications([]), []);

    const value = useMemo(() => ({ notifications, addNotification, clearNotifications }),
        [notifications, addNotification, clearNotifications]);

    return (
        <NotificationContext.Provider value={value}>
            {children}
        </NotificationContext.Provider>
    );
};

export const useNotification = () => useContext(NotificationContext);

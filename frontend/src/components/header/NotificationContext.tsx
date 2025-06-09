import React, { createContext, useContext, useState } from 'react';

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

    const addNotification = (message: string) => {
        setNotifications((prev) => [
            ...prev,
            { id: crypto.randomUUID(), message },
        ]);
    };

    const clearNotifications = () => setNotifications([]);

    return (
        <NotificationContext.Provider value={{ notifications, addNotification, clearNotifications }}>
            {children}
        </NotificationContext.Provider>
    );
};

export const useNotification = () => useContext(NotificationContext);

import React, { useState } from 'react';
import {
    IconButton,
    Badge,
    Menu,
    MenuItem,
    ListItemText,
} from '@mui/material';
import NotificationsIcon from '@mui/icons-material/Notifications';
import {useTranslation} from "react-i18next";
import {useNotification} from "./NotificationContext";

export default function NotificationBell() {
    const { t } = useTranslation();
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);

    const { notifications, clearNotifications } = useNotification();

    const handleOpen = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };

    const handleClose = () => {
        setAnchorEl(null);
    };


    return (
        <>
            <IconButton onClick={handleOpen} color="inherit">
                <Badge badgeContent={notifications.length} color="error">
                    <NotificationsIcon />
                </Badge>
            </IconButton>

            <Menu anchorEl={anchorEl} open={open} onClose={handleClose}>
                {notifications.length === 0 ? (
                    <MenuItem disabled>{t("header.notifications.none")}</MenuItem>
                ) : (
                    notifications.map((note) => (
                        <MenuItem key={note.id} onClick={handleClose}>
                            <ListItemText primary={note.message} />
                        </MenuItem>
                    ))
                )}
                {notifications.length > 0 && (
                    <MenuItem onClick={() => { clearNotifications(); handleClose(); }}>
                        <ListItemText primary={ t('header.notifications.deleteall')} />
                    </MenuItem>
                )}
            </Menu>
        </>
    );
}

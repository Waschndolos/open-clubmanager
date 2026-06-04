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
import {useTheme} from "@mui/material/styles";
import {useNotification} from "../../context/NotificationContext";

export default function NotificationBell() {
    const { t } = useTranslation();
    const theme = useTheme();
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
            <IconButton
                onClick={handleOpen}
                color="inherit"
                sx={{
                    transition: 'all 0.3s ease',
                    '&:hover': {
                        backgroundColor: theme.palette.mode === 'dark'
                            ? 'rgba(0, 255, 194, 0.1)'
                            : 'rgba(0, 200, 154, 0.1)',
                    }
                }}
            >
                <Badge badgeContent={notifications.length} color="error">
                    <NotificationsIcon />
                </Badge>
            </IconButton>

            <Menu
                anchorEl={anchorEl}
                open={open}
                onClose={handleClose}
                slotProps={{
                    paper: {
                        sx: {
                            background: theme.palette.mode === 'dark'
                                ? 'rgba(26, 32, 34, 0.95)'
                                : 'rgba(255, 255, 255, 0.95)',
                            backdropFilter: 'blur(12px)',
                            border: theme.palette.mode === 'dark'
                                ? '1px solid rgba(0, 255, 194, 0.1)'
                                : '1px solid rgba(0, 200, 154, 0.12)',
                        }
                    }
                }}
            >
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

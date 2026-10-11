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
import {useNotification} from "../../context/NotificationContext";
import {useNavigate} from "react-router-dom";
import {useCollaboration} from "../../context/CollaborationContext";

export default function NotificationBell() {
    const { t } = useTranslation();
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);

    const { notifications, clearNotifications } = useNotification();
    const navigate = useNavigate();
    const { status } = useCollaboration();

    // Collaboration items link to where they are resolved (UX §3).
    const collabItems: { id: string; message: string; link: string }[] = [];
    if (status && status.mode !== 'local') {
        if (status.pendingRequests > 0) {
            collabItems.push({ id: 'requests', message: t('collaboration.popover.requests', { count: status.pendingRequests }), link: '/collaboration?tab=people' });
        }
        if (status.conflicts > 0) {
            collabItems.push({ id: 'conflicts', message: t('collaboration.popover.conflicts', { count: status.conflicts }), link: '/collaboration/conflicts' });
        }
        if (status.locked) {
            collabItems.push({ id: 'locked', message: t('collaboration.status.locked'), link: '/collaboration' });
        }
        if (status.mode === 'revoked') {
            collabItems.push({ id: 'revoked', message: t('collaboration.status.revoked'), link: '/collaboration' });
        }
        if (status.lostChanges > 0) {
            collabItems.push({ id: 'lost', message: t('collaboration.popover.lostChanges'), link: '/collaboration?tab=people' });
        }
        if (status.badKeys > 0) {
            collabItems.push({ id: 'keys', message: t('collaboration.popover.badKeys'), link: '/collaboration' });
        }
    }
    const count = notifications.length + collabItems.length;

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
                size="small"
                aria-label={t('header.notifications.title')}
            >
                <Badge badgeContent={count} color="error">
                    <NotificationsIcon fontSize="small" />
                </Badge>
            </IconButton>

            <Menu
                anchorEl={anchorEl}
                open={open}
                onClose={handleClose}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                transformOrigin={{ vertical: 'top', horizontal: 'right' }}
            >
                {collabItems.map((item) => (
                    <MenuItem key={item.id} onClick={() => { handleClose(); navigate(item.link); }} sx={{ whiteSpace: 'normal', maxWidth: 360 }}>
                        <ListItemText primary={item.message} />
                    </MenuItem>
                ))}
                {count === 0 ? (
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

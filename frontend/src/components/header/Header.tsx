import {AppBar, Box, IconButton, Toolbar, Typography} from '@mui/material'
import React from "react";
import NotificationBell from "./NotificationBell";
import MenuIcon from "@mui/icons-material/Menu";
import ThemeToggle from "./ThemeToggle"; // Import the new ThemeToggle

// HeaderProps type
type HeaderProps = {
    sidebarCollapsed: boolean;
    onToggleSidebar: () => void;
};

export default function Header({onToggleSidebar }: HeaderProps) {
    return (
        <AppBar position="static" color="primary" sx={{
            backgroundColor: (theme) => theme.palette.mode === 'dark' ? '#23243a' : theme.palette.primary.main,
            boxShadow: (theme) => theme.palette.mode === 'dark' ? '0 2px 8px 0 rgba(142,202,230,0.10)' : '0 2px 8px 0 rgba(0,0,0,0.10)',
            transition: 'background 0.3s',
        }}>
            <Toolbar>
                <IconButton
                    color="inherit"
                    edge="start"
                    onClick={onToggleSidebar}
                    sx={{ mr: 2 }}
                    aria-label="toggle sidebar"
                >
                    <MenuIcon />
                </IconButton>
                <Typography variant="h6" sx={{flexGrow: 1, paddingLeft: 2}}>
                    Open ClubManager
                </Typography>

                <Box display="flex" alignItems="center" gap={1}>
                    {/* Theme toggle button (sun/moon) */}
                    <ThemeToggle />
                    {/* Notification bell */}
                    <NotificationBell/>
                </Box>
            </Toolbar>
        </AppBar>
    )
}

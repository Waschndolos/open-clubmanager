import {AppBar, Box, IconButton, Toolbar, Typography} from '@mui/material'
import React from "react";
import NotificationBell from "./NotificationBell";
import MenuIcon from "@mui/icons-material/Menu";
import ThemeToggle from "./ThemeToggle";
import LogoutButton from "./LogoutButton";
import { useTheme } from '@mui/material/styles';

// HeaderProps type
type HeaderProps = {
    sidebarCollapsed: boolean;
    onToggleSidebar: () => void;
};

export default function Header({onToggleSidebar }: HeaderProps) {
    const theme = useTheme();
    
    return (
        <AppBar position="static" elevation={0}>
            <Toolbar
                variant="dense"
                sx={{
                    minHeight: 56,
                    px: { xs: 1.5, sm: 2 },
                    gap: 1.5,
                }}
            >
                <IconButton
                    color="inherit"
                    edge="start"
                    onClick={onToggleSidebar}
                    size="small"
                    aria-label="toggle sidebar"
                >
                    <MenuIcon fontSize="small" />
                </IconButton>

                <Typography
                    variant="h6"
                    sx={{
                        flexGrow: 1,
                        fontWeight: 700,
                        letterSpacing: '-0.02em',
                        fontSize: '1.05rem',
                    }}
                >
                    Open ClubManager
                </Typography>

                <Box
                    sx={{
                        display: "flex",
                        alignItems: "center",
                        gap: 0.5,
                        bgcolor: theme.palette.mode === 'dark'
                            ? 'rgba(255, 255, 255, 0.04)'
                            : 'rgba(0, 0, 0, 0.03)',
                        border: `1px solid ${theme.palette.mode === 'dark'
                            ? 'rgba(255, 255, 255, 0.06)'
                            : 'rgba(0, 0, 0, 0.06)'}`,
                        borderRadius: 3,
                        p: 0.5,
                    }}
                >
                    <ThemeToggle />
                    <NotificationBell />
                    <LogoutButton />
                </Box>
            </Toolbar>
        </AppBar>
    )
}

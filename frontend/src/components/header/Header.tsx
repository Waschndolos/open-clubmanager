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
        <AppBar position="static" sx={{
            backgroundColor: theme.palette.mode === 'dark'
                ? 'rgba(26, 32, 34, 0.95)'
                : 'rgba(255, 255, 255, 0.95)',
            backdropFilter: 'blur(20px)',
            borderBottom: theme.palette.mode === 'dark'
                ? '1px solid rgba(0, 255, 194, 0.12)'
                : '1px solid rgba(0, 200, 154, 0.15)',
            boxShadow: '0 4px 12px rgba(0, 0, 0, 0.08)',
            borderRadius: 0,
        }}>
            <Toolbar sx={{ py: 1 }}>
                <IconButton
                    color="inherit"
                    edge="start"
                    onClick={onToggleSidebar}
                    sx={{
                        mr: 2,
                        transition: 'all 0.3s ease',
                        '&:hover': {
                            backgroundColor: theme.palette.mode === 'dark'
                                ? 'rgba(0, 255, 194, 0.1)'
                                : 'rgba(0, 200, 154, 0.1)',
                        }
                    }}
                    aria-label="toggle sidebar"
                >
                    <MenuIcon />
                </IconButton>
                <Typography
                    variant="h6"
                    sx={{
                        flexGrow: 1,
                        paddingLeft: 2,
                        fontWeight: 700,
                        letterSpacing: '-0.01em',
                    }}
                >
                    Open ClubManager
                </Typography>

                <Box display="flex" alignItems="center" gap={1}>
                    <ThemeToggle />
                    <NotificationBell/>
                    <LogoutButton/>
                </Box>
            </Toolbar>
        </AppBar>
    )
}

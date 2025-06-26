import {AppBar, Box, IconButton, Toolbar, Typography} from '@mui/material'
import {DarkMode, LightMode} from '@mui/icons-material'
import {useThemeContext} from '../../theme/ThemeContext'
import React from "react";
import NotificationBell from "./NotificationBell";
import MenuIcon from "@mui/icons-material/Menu";

type HeaderProps = {
    sidebarCollapsed: boolean;
    onToggleSidebar: () => void;
};

export default function Header({onToggleSidebar }: HeaderProps) {
    const {mode, toggleTheme} = useThemeContext()

    return (
        <AppBar position="static" color="primary">
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

                    <IconButton onClick={toggleTheme} color={"secondary"}>
                        {mode === 'light' ? <DarkMode/> : <LightMode/>}
                    </IconButton>
                    <NotificationBell/>
                </Box>
            </Toolbar>
        </AppBar>
    )
}

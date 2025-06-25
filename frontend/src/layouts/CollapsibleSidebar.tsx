import React, { useState } from 'react';
import { Box, IconButton, List, ListItemButton, ListItemIcon, ListItemText, Tooltip } from '@mui/material';
import { Menu, Dashboard, Person, Settings } from '@mui/icons-material';

const menuItems = [
    { label: 'Dashboard', icon: <Dashboard />, link: '/dashboard' },
    { label: 'Members', icon: <Person />, link: '/members' },
    { label: 'Settings', icon: <Settings />, link: '/settings' },
];

export default function CollapsibleSidebar() {
    const [isCollapsed, setIsCollapsed] = useState(false);

    return (
        <Box sx={{ display: 'flex', height: '100vh', borderRight: '1px solid #ddd' }}>
            {/* Sidebar */}
            <Box
                sx={{
                    width: isCollapsed ? 60 : 240,
                    transition: 'width 0.3s',
                    bgcolor: 'background.paper',
                    overflow: 'hidden',
                }}
            >
                {/* Toggle Button */}
                <IconButton onClick={() => setIsCollapsed(!isCollapsed)} sx={{ m: 1 }}>
                    <Menu />
                </IconButton>

                <List>
                    {menuItems.map(({ label, icon, link }) => (
                        <ListItemButton key={label} sx={{ justifyContent: isCollapsed ? 'center' : 'flex-start' }}>
                            <Tooltip title={isCollapsed ? label : ''} placement="right">
                                <ListItemIcon sx={{ minWidth: 0, justifyContent: 'center' }}>
                                    {icon}
                                </ListItemIcon>
                            </Tooltip>
                            {!isCollapsed && <ListItemText primary={label} sx={{ ml: 2 }} />}
                        </ListItemButton>
                    ))}
                </List>
            </Box>

            {/* Content-Bereich */}
            <Box sx={{ flexGrow: 1, p: 2 }}>
                {/* Hier kommt dein Hauptcontent */}
                Content goes here
            </Box>
        </Box>
    );
}

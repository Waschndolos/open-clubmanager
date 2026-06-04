import {Box} from '@mui/material';
import AppMenu from "../components/menu/AppMenu";
import Content from "../components/content/Content";
import Header from "../components/header/Header";
import {useState} from "react";


export default function MainLayout() {
    const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

    return (
        <Box 
            sx={{ display: "flex", flexDirection: "column", height: "100vh", width: "100vw" }}
        >
            <Header
                sidebarCollapsed={sidebarCollapsed}
                onToggleSidebar={() => setSidebarCollapsed((c) => !c)}
            />
            <Box sx={{ display: "flex", flexGrow: 1, overflow: "hidden" }}>
                <Box
                    sx={{
                        width: sidebarCollapsed ? 60 : "20%",
                        minWidth: sidebarCollapsed ? 60 : "20%",
                        borderRight: (theme) => theme.custom.border,
                        bgcolor: (theme) => theme.palette.background.paper,
                        transition: "width 0.3s cubic-bezier(0.4, 0, 0.2, 1), background-color 0.3s ease",
                        overflowY: "auto",
                        overflowX: "hidden",
                        scrollbarWidth: "thin",
                        '&::-webkit-scrollbar': {
                            width: '6px',
                        },
                        '&::-webkit-scrollbar-track': {
                            background: 'transparent',
                        },
                        '&::-webkit-scrollbar-thumb': {
                            backgroundColor: (theme) => theme.palette.mode === 'dark' 
                                ? 'rgba(0, 255, 194, 0.2)' 
                                : 'rgba(0, 200, 154, 0.2)',
                            borderRadius: '3px',
                        },
                    }}
                >
                    <AppMenu collapsed={sidebarCollapsed} />
                </Box>
                <Box 
                    className="transition-colors duration-500"
                    sx={{ 
                        flexGrow: 1, 
                        overflow: "auto",
                        scrollbarWidth: "thin",
                        '&::-webkit-scrollbar': {
                            width: '8px',
                        },
                        '&::-webkit-scrollbar-track': {
                            background: 'transparent',
                        },
                        '&::-webkit-scrollbar-thumb': {
                            backgroundColor: (theme) => theme.palette.mode === 'dark' 
                                ? 'rgba(0, 255, 194, 0.15)' 
                                : 'rgba(0, 200, 154, 0.15)',
                            borderRadius: '4px',
                        },
                    }}
                >
                    <Content />
                </Box>
            </Box>
        </Box>
    );
}

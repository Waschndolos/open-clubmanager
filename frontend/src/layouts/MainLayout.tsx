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
                        borderRight: "1px solid #ddd",
                        transition: "width 0.3s, min-width 0.3s",
                        overflowY: "auto",
                    }}
                >
                    <AppMenu collapsed={sidebarCollapsed} />
                </Box>
                <Box 
                    className="transition-colors duration-500"
                    sx={{ flexGrow: 1, overflow: "auto" }}
                >
                    <Content />
                </Box>
            </Box>
        </Box>
    );
}

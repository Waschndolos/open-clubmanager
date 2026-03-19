import {Box} from '@mui/material';
import AppMenu from "../components/menu/AppMenu";
import Content from "../components/content/Content";
import Header from "../components/header/Header";
import ReadOnlyBanner from "../components/common/ReadOnlyBanner";
import {useState} from "react";
import { useStorage } from "../context/StorageContext";
import { isElectronFolderMode } from "../lib/environment";


export default function MainLayout() {
    const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
    const { status, requestEditMode } = useStorage();

    const showReadOnlyBanner =
        isElectronFolderMode() && status?.mode === 'readonly';

    return (
        <Box 
            sx={{ display: "flex", flexDirection: "column", height: "100vh", width: "100vw" }}
        >
            <Header
                sidebarCollapsed={sidebarCollapsed}
                onToggleSidebar={() => setSidebarCollapsed((c) => !c)}
            />
            {showReadOnlyBanner && status && (
                <ReadOnlyBanner status={status} onRequestEditMode={requestEditMode} />
            )}
            <Box sx={{ display: "flex", flexGrow: 1, overflow: "hidden" }}>
                <Box
                    sx={{
                        width: sidebarCollapsed ? 60 : "20%",
                        minWidth: sidebarCollapsed ? 60 : "20%",
                        borderRight: (theme) => theme.custom.border,
                        bgcolor: (theme) =>
                            theme.palette.mode === 'dark' ? '#14161F' : '#ffffff',
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

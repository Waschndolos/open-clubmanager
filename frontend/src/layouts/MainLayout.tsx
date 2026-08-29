import { Box } from "@mui/material";
import AppMenu from "../components/menu/AppMenu";
import Content from "../components/content/Content";
import Header from "../components/header/Header";
import { useState } from "react";

const SIDEBAR_WIDTH = 220;
const SIDEBAR_COLLAPSED_WIDTH = 64;

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
            width: sidebarCollapsed ? SIDEBAR_COLLAPSED_WIDTH : SIDEBAR_WIDTH,
            minWidth: sidebarCollapsed ? SIDEBAR_COLLAPSED_WIDTH : SIDEBAR_WIDTH,
            borderRight: (theme) => theme.custom.border,
            bgcolor: (theme) => theme.palette.background.paper,
            transition:
              "width 0.25s cubic-bezier(0.4, 0, 0.2, 1), background-color 0.3s ease",
            overflowY: "auto",
            overflowX: "hidden",
            scrollbarWidth: "thin",
            '&::-webkit-scrollbar': {
              width: "4px",
            },
            '&::-webkit-scrollbar-track': {
              background: "transparent",
            },
            '&::-webkit-scrollbar-thumb': {
              backgroundColor: (theme) =>
                theme.palette.mode === "dark"
                  ? "rgba(255, 255, 255, 0.08)"
                  : "rgba(0, 0, 0, 0.08)",
              borderRadius: "2px",
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
              width: "6px",
            },
            '&::-webkit-scrollbar-track': {
              background: "transparent",
            },
            '&::-webkit-scrollbar-thumb': {
              backgroundColor: (theme) =>
                theme.palette.mode === "dark"
                  ? "rgba(255, 255, 255, 0.08)"
                  : "rgba(0, 0, 0, 0.08)",
              borderRadius: "3px",
            },
          }}
        >
          <Content />
        </Box>
      </Box>
    </Box>
  );
}

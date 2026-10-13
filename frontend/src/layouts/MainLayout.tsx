import { Box, Drawer, useMediaQuery } from "@mui/material";
import { useTheme } from "@mui/material/styles";
import { useState } from "react";
import AppMenu from "../components/menu/AppMenu";
import Content from "../components/content/Content";
import Header from "../components/header/Header";
import { useAuth } from "../context/AuthContext";
import { useAttendanceSync } from "../pages/attendance/useAttendanceSync";

/** Sends attendance recorded offline once the server is reachable again. */
function AttendanceSync() {
  useAttendanceSync();
  return null;
}

const SIDEBAR_WIDTH = 248;
const SIDEBAR_COLLAPSED_WIDTH = 72;
const COLLAPSED_KEY = "sidebarCollapsed";

function readCollapsed(): boolean {
  try {
    return localStorage.getItem(COLLAPSED_KEY) === "1";
  } catch {
    return false;
  }
}

export default function MainLayout() {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const [collapsed, setCollapsed] = useState(readCollapsed);
  const [drawerOpen, setDrawerOpen] = useState(false);

  const toggleCollapsed = () => {
    setCollapsed((c) => {
      try {
        localStorage.setItem(COLLAPSED_KEY, c ? "0" : "1");
      } catch {
        // Remembering the sidebar state is a convenience only.
      }
      return !c;
    });
  };

  const width = collapsed ? SIDEBAR_COLLAPSED_WIDTH : SIDEBAR_WIDTH;
  const { appRole } = useAuth();

  return (
    <Box sx={{ display: "flex", height: "100vh", width: "100%", bgcolor: "background.default" }}>
      {appRole && <AttendanceSync />}
      {isMobile ? (
        <Drawer
          open={drawerOpen}
          onClose={() => setDrawerOpen(false)}
          ModalProps={{ keepMounted: true }}
          slotProps={{ paper: { sx: { width: SIDEBAR_WIDTH + 16, borderRadius: 0, border: 0 } } }}
        >
          <AppMenu collapsed={false} onNavigate={() => setDrawerOpen(false)} />
        </Drawer>
      ) : (
        <Box
          component="aside"
          sx={{
            width,
            flexShrink: 0,
            borderRight: 1,
            borderColor: "divider",
            bgcolor: "background.paper",
            transition: "width 0.2s cubic-bezier(0.4, 0, 0.2, 1)",
            overflow: "hidden",
          }}
        >
          <AppMenu collapsed={collapsed} onToggleCollapsed={toggleCollapsed} />
        </Box>
      )}

      <Box sx={{ flexGrow: 1, minWidth: 0, overflowY: "auto", scrollbarWidth: "thin" }}>
        <Header onOpenNavigation={isMobile ? () => setDrawerOpen(true) : undefined} />
        <Content />
      </Box>
    </Box>
  );
}

import { Box, List, Typography } from "@mui/material";
import AppMenuItem from "./AppMenuItem";
import packageJson from "../../../package.json";
import { useTranslation } from "react-i18next";
import {
  AccountBalance,
  Badge,
  GridView,
  History,
  Inventory2,
  ManageAccounts,
  People,
  Settings,
} from "@mui/icons-material";
import { useAuth } from "../../context/AuthContext";

type Props = {
  collapsed: boolean;
};

export default function AppMenu({ collapsed }: Props) {
  const { t } = useTranslation();
  const { appRole } = useAuth();

  const isAdmin = appRole === "ADMIN";
  const canViewFinance = appRole === "ADMIN" || appRole === "TREASURER";

  return (
    <Box
      sx={{
        display: "flex",
        flexDirection: "column",
        height: "100%",
        width: "100%",
        py: 1.5,
        px: collapsed ? 1 : 1.5,
      }}
    >
      <List sx={{ flexGrow: 1, display: "flex", flexDirection: "column", gap: 0.25 }}>
        <AppMenuItem
          label={t("menu.dashboard")}
          icon={<GridView fontSize="small" />}
          link="dashboard"
          collapsed={collapsed}
        />
        <AppMenuItem
          label={t("menu.members")}
          icon={<ManageAccounts fontSize="small" />}
          link="members"
          collapsed={collapsed}
        />
        {canViewFinance && (
          <AppMenuItem
            label={t("menu.finance")}
            icon={<AccountBalance fontSize="small" />}
            link="finance"
            collapsed={collapsed}
          />
        )}
        <AppMenuItem
          label={t("menu.inventory")}
          icon={<Inventory2 fontSize="small" />}
          link="inventory"
          collapsed={collapsed}
        />
        <AppMenuItem
          label={t("menu.entities")}
          icon={<Badge fontSize="small" />}
          link="entities"
          collapsed={collapsed}
        />
        <AppMenuItem
          label={t("menu.history")}
          icon={<History fontSize="small" />}
          link="history"
          collapsed={collapsed}
        />
        {isAdmin && (
          <AppMenuItem
            label={t("menu.users")}
            icon={<People fontSize="small" />}
            link="users"
            collapsed={collapsed}
          />
        )}
        <AppMenuItem
          label={t("menu.settings")}
          icon={<Settings fontSize="small" />}
          link="settings"
          collapsed={collapsed}
        />
      </List>

      <Box sx={{ marginTop: "auto", textAlign: "center", py: 1.5, px: 1 }}>
        {collapsed ? (
          <Typography
            variant="caption"
            color="text.secondary"
            sx={{ fontSize: "0.6rem", fontWeight: 600, opacity: 0.7 }}
          >
            v{packageJson.version}
          </Typography>
        ) : (
          <Typography
            variant="caption"
            color="text.secondary"
            noWrap
            sx={{
              fontSize: "0.7rem",
              fontWeight: 600,
              letterSpacing: "0.02em",
              opacity: 0.6,
            }}
          >
            v{packageJson.version}
          </Typography>
        )}
      </Box>
    </Box>
  );
}
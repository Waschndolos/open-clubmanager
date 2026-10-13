import { ReactElement } from "react";
import { useTranslation } from "react-i18next";
import AccountBalanceOutlined from "@mui/icons-material/AccountBalanceOutlined";
import AccountTreeOutlined from "@mui/icons-material/AccountTreeOutlined";
import GroupsOutlined from "@mui/icons-material/GroupsOutlined";
import FamilyRestroomOutlined from "@mui/icons-material/FamilyRestroomOutlined";
import EventOutlined from "@mui/icons-material/EventOutlined";
import FactCheckOutlined from "@mui/icons-material/FactCheckOutlined";
import CelebrationOutlined from "@mui/icons-material/CelebrationOutlined";
import TaskAltOutlined from "@mui/icons-material/TaskAltOutlined";
import HistoryOutlined from "@mui/icons-material/HistoryOutlined";
import HubOutlined from "@mui/icons-material/HubOutlined";
import Inventory2Outlined from "@mui/icons-material/Inventory2Outlined";
import ManageAccountsOutlined from "@mui/icons-material/ManageAccountsOutlined";
import WorkspacePremiumOutlined from "@mui/icons-material/WorkspacePremiumOutlined";
import SettingsOutlined from "@mui/icons-material/SettingsOutlined";
import SpaceDashboardOutlined from "@mui/icons-material/SpaceDashboardOutlined";
import { useAuth } from "../../context/AuthContext";
import { usePermission } from "../../context/CollaborationContext";

export type NavItem = {
    label: string;
    icon: ReactElement;
    link: string;
};

export type NavGroup = {
    /** Group heading; the first group has none. */
    label?: string;
    items: NavItem[];
};

/** Navigation entries visible to the current login, grouped for the sidebar. */
export function useNavGroups(): NavGroup[] {
    const { t } = useTranslation();
    const { appRole } = useAuth();

    const isAdmin = appRole === "ADMIN";
    // The device's role in a collaboration club limits what this login sees (UX F10).
    const financeArea = usePermission("finance");
    const inventoryArea = usePermission("inventory");
    const membersArea = usePermission("membersCore");
    const canViewFinance = (appRole === "ADMIN" || appRole === "TREASURER") && financeArea !== "none";

    const club: NavItem[] = [
        { label: t("nav.members"), icon: <GroupsOutlined />, link: "/members" },
        ...(membersArea !== "none" ? [{ label: t("nav.events"), icon: <EventOutlined />, link: "/events" }] : []),
        ...(membersArea !== "none" ? [{ label: t("nav.attendance"), icon: <FactCheckOutlined />, link: "/attendance" }] : []),
        ...(membersArea !== "none" ? [{ label: t("nav.households"), icon: <FamilyRestroomOutlined />, link: "/households" }] : []),
        ...(canViewFinance ? [{ label: t("nav.finance"), icon: <AccountBalanceOutlined />, link: "/finance" }] : []),
        ...(inventoryArea !== "none" ? [{ label: t("nav.inventory"), icon: <Inventory2Outlined />, link: "/inventory" }] : []),
        ...(membersArea !== "none" ? [{ label: t("nav.honors"), icon: <CelebrationOutlined />, link: "/honors" }] : []),
        ...(membersArea !== "none" && (appRole === "ADMIN" || appRole === "SECRETARY")
            ? [{ label: t("nav.qualifications"), icon: <WorkspacePremiumOutlined />, link: "/qualifications" }] : []),
        ...(membersArea !== "none" ? [{ label: t("nav.tasks"), icon: <TaskAltOutlined />, link: "/tasks" }] : []),
        { label: t("nav.entities"), icon: <AccountTreeOutlined />, link: "/entities" },
    ];

    const admin: NavItem[] = [
        { label: t("nav.history"), icon: <HistoryOutlined />, link: "/history" },
        ...(isAdmin ? [{ label: t("nav.users"), icon: <ManageAccountsOutlined />, link: "/users" }] : []),
        { label: t("nav.collaboration"), icon: <HubOutlined />, link: "/collaboration" },
        { label: t("nav.settings"), icon: <SettingsOutlined />, link: "/settings" },
    ];

    return [
        { items: [{ label: t("nav.dashboard"), icon: <SpaceDashboardOutlined />, link: "/dashboard" }] },
        { label: t("nav.club"), items: club },
        { label: t("nav.admin"), items: admin },
    ];
}

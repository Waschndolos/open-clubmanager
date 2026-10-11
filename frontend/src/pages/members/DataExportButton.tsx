import {useState} from "react";
import {IconButton, Menu, MenuItem, Tooltip} from "@mui/material";
import PrivacyTipOutlined from "@mui/icons-material/PrivacyTipOutlined";
import {useTranslation} from "react-i18next";
import {Member} from "../../api/types";
import {DataExportFormat, downloadMemberDataExport} from "../../api/dataExport";
import {useAuth} from "../../context/AuthContext";
import {usePermission} from "../../context/CollaborationContext";
import {useNotification} from "../../context/NotificationContext";

/** "Create data export" (GDPR Art. 15/20) for the selected member, as PDF or JSON. */
export default function DataExportButton({member}: {member: Member | null}) {
    const {t} = useTranslation();
    const {appRole} = useAuth();
    const {addNotification} = useNotification();
    const [anchor, setAnchor] = useState<HTMLElement | null>(null);
    // The report covers every area, so a device needs access to all of them (the server checks too).
    const areas = [usePermission("membersCore"), usePermission("membersContact"), usePermission("membersBank"),
        usePermission("finance"), usePermission("inventory")];

    if ((appRole !== "ADMIN" && appRole !== "SECRETARY") || areas.includes("none")) return null;

    const run = async (format: DataExportFormat) => {
        setAnchor(null);
        if (!member) return;
        try {
            await downloadMemberDataExport(member.id, format);
        } catch {
            addNotification(t("dataExport.failed"));
        }
    };

    return (
        <>
            <Tooltip title={t("dataExport.create")}>
                <span>
                    <IconButton color="primary" disabled={!member} aria-label={t("dataExport.create")}
                                aria-haspopup="menu" onClick={(e) => setAnchor(e.currentTarget)}>
                        <PrivacyTipOutlined/>
                    </IconButton>
                </span>
            </Tooltip>
            <Menu anchorEl={anchor} open={!!anchor} onClose={() => setAnchor(null)}>
                <MenuItem onClick={() => run("pdf")}>{t("dataExport.pdf")}</MenuItem>
                <MenuItem onClick={() => run("json")}>{t("dataExport.json")}</MenuItem>
            </Menu>
        </>
    );
}

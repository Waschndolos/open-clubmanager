import React, { useEffect, useState } from "react";
import { Avatar, Box, Paper, Typography } from "@mui/material";
import AccountCircleIcon from "@mui/icons-material/AccountCircle";
import { useAuth } from "../../context/AuthContext";
import { getProfile } from "../../api/authentication";
import PageHeader from "../../components/common/PageHeader";
import { useTranslation } from "react-i18next";

export default function Profile() {
    const { t } = useTranslation();
    const { accessToken } = useAuth();
    const [email, setEmail] = useState("");

    useEffect(() => {
        if (accessToken) {
            getProfile(accessToken).then(data => setEmail(data.email));
        }
    }, [accessToken]);

    if (!accessToken) {
        return (
            <Box sx={{ p: 4 }}>
                <Typography color="text.secondary">{t("login.title")}</Typography>
            </Box>
        );
    }

    const initials = email
        ? email.charAt(0).toUpperCase()
        : "?";

    return (
        <Box sx={{ p: 4 }}>
            <PageHeader
                title={t("menu.profile")}
                icon={<AccountCircleIcon fontSize="small" />}
            />
            <Paper sx={{ p: 4, maxWidth: 480 }}>
                <Box sx={{ display: "flex", alignItems: "center", gap: 3 }}>
                    <Avatar
                        sx={{
                            width: 72,
                            height: 72,
                            bgcolor: "primary.main",
                            fontSize: "2rem",
                            fontWeight: 700,
                        }}
                    >
                        {initials}
                    </Avatar>
                    <Box>
                        <Typography variant="h6" fontWeight={700}>
                            {email}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                            {t("profile.role")}
                        </Typography>
                    </Box>
                </Box>
            </Paper>
        </Box>
    );
}

import React, { useEffect, useState } from "react";
import { Alert, Avatar, Box, Button, Paper, Snackbar, TextField, Typography } from "@mui/material";
import AccountCircleIcon from "@mui/icons-material/AccountCircle";
import LockIcon from "@mui/icons-material/Lock";
import { useAuth } from "../../context/AuthContext";
import { getProfile } from "../../api/authentication";
import { changePassword } from "../../api/users";
import PageHeader from "../../components/common/PageHeader";
import { useTranslation } from "react-i18next";

export default function Profile() {
    const { t } = useTranslation();
    const { accessToken, appRole } = useAuth();
    const [email, setEmail] = useState("");

    const [currentPassword, setCurrentPassword] = useState("");
    const [newPassword, setNewPassword] = useState("");
    const [confirmPassword, setConfirmPassword] = useState("");
    const [pwLoading, setPwLoading] = useState(false);
    const [snack, setSnack] = useState<{ open: boolean; message: string; severity: "success" | "error" }>({
        open: false,
        message: "",
        severity: "success",
    });

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

    const initials = email ? email.charAt(0).toUpperCase() : "?";

    const handleChangePassword = async () => {
        if (newPassword.length < 8) {
            setSnack({ open: true, message: t("profile.changePassword.tooShort"), severity: "error" });
            return;
        }
        if (newPassword !== confirmPassword) {
            setSnack({ open: true, message: t("profile.changePassword.mismatch"), severity: "error" });
            return;
        }
        setPwLoading(true);
        try {
            await changePassword(newPassword, currentPassword || undefined);
            setCurrentPassword("");
            setNewPassword("");
            setConfirmPassword("");
            setSnack({ open: true, message: t("profile.changePassword.success"), severity: "success" });
        } catch {
            setSnack({ open: true, message: t("profile.changePassword.error"), severity: "error" });
        } finally {
            setPwLoading(false);
        }
    };

    return (
        <Box sx={{ p: 4 }}>
            <PageHeader
                title={t("menu.profile")}
                icon={<AccountCircleIcon fontSize="small" />}
            />

            <Paper sx={{ p: 4, maxWidth: 480, mb: 3 }}>
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
                            {appRole ? t(`users.roles.${appRole}`) : t("profile.role")}
                        </Typography>
                    </Box>
                </Box>
            </Paper>

            <Paper sx={{ p: 4, maxWidth: 480 }}>
                <Box sx={{ display: "flex", alignItems: "center", gap: 1, mb: 3 }}>
                    <LockIcon color="action" fontSize="small" />
                    <Typography variant="h6" fontWeight={700}>
                        {t("profile.changePassword.title")}
                    </Typography>
                </Box>
                <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                    <TextField
                        label={t("profile.changePassword.currentPassword")}
                        type="password"
                        value={currentPassword}
                        onChange={(e) => setCurrentPassword(e.target.value)}
                        fullWidth
                        size="small"
                    />
                    <TextField
                        label={t("profile.changePassword.newPassword")}
                        type="password"
                        value={newPassword}
                        onChange={(e) => setNewPassword(e.target.value)}
                        fullWidth
                        size="small"
                        required
                    />
                    <TextField
                        label={t("profile.changePassword.confirmPassword")}
                        type="password"
                        value={confirmPassword}
                        onChange={(e) => setConfirmPassword(e.target.value)}
                        fullWidth
                        size="small"
                        required
                    />
                    <Button
                        variant="contained"
                        onClick={handleChangePassword}
                        disabled={pwLoading || !newPassword || !confirmPassword}
                        sx={{ alignSelf: "flex-start" }}
                    >
                        {pwLoading ? t("profile.changePassword.saving") : t("profile.changePassword.save")}
                    </Button>
                </Box>
            </Paper>

            <Snackbar
                open={snack.open}
                autoHideDuration={4000}
                onClose={() => setSnack((s) => ({ ...s, open: false }))}
            >
                <Alert severity={snack.severity} onClose={() => setSnack((s) => ({ ...s, open: false }))}>
                    {snack.message}
                </Alert>
            </Snackbar>
        </Box>
    );
}

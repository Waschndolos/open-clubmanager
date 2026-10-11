import { Box, Typography } from "@mui/material";
import { alpha } from "@mui/material/styles";
import { ReactNode } from "react";

interface PageHeaderProps {
    title: string;
    subtitle?: string;
    icon?: ReactNode;
    actions?: ReactNode;
}

/**
 * Page title row: optional icon badge, title and subtitle on the left, actions on the right.
 * Actions wrap below the title on narrow screens.
 */
export default function PageHeader({ title, subtitle, icon, actions }: PageHeaderProps) {
    return (
        <Box
            sx={{
                display: "flex",
                alignItems: { xs: "flex-start", sm: "center" },
                justifyContent: "space-between",
                mb: { xs: 3, sm: 4 },
                flexWrap: "wrap",
                gap: 2,
            }}
        >
            <Box sx={{ display: "flex", alignItems: "center", gap: 1.75, minWidth: 0 }}>
                {icon && (
                    <Box
                        aria-hidden
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                            width: 44,
                            height: 44,
                            borderRadius: 2.5,
                            flexShrink: 0,
                            color: "primary.main",
                            bgcolor: (theme) => alpha(theme.palette.primary.main, 0.1),
                            "& .MuiSvgIcon-root": { fontSize: 24 },
                        }}
                    >
                        {icon}
                    </Box>
                )}
                <Box sx={{ minWidth: 0 }}>
                    <Typography variant="h4" component="h1" sx={{ lineHeight: 1.15, fontSize: { xs: "1.5rem", sm: "1.875rem" } }}>
                        {title}
                    </Typography>
                    {subtitle && (
                        <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                            {subtitle}
                        </Typography>
                    )}
                </Box>
            </Box>
            {actions && (
                <Box sx={{ display: "flex", alignItems: "center", gap: 1, ml: { sm: "auto" }, flexWrap: "wrap" }}>
                    {actions}
                </Box>
            )}
        </Box>
    );
}

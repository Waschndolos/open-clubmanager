import { Box, Typography } from "@mui/material";
import { ReactNode } from "react";

interface PageHeaderProps {
    title: string;
    subtitle?: string;
    icon?: ReactNode;
    actions?: ReactNode;
}

/**
 * Reusable page header component with title, optional subtitle, optional icon and optional actions slot.
 */
export default function PageHeader({ title, subtitle, icon, actions }: PageHeaderProps) {
    return (
        <Box
            sx={{
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                mb: 3,
                flexWrap: "wrap",
                gap: 2,
            }}
        >
            <Box sx={{ display: "flex", alignItems: "center", gap: 1.5 }}>
                {icon && (
                    <Box
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                            color: "primary.main",
                            flexShrink: 0,
                            '& .MuiSvgIcon-root': { fontSize: '1.75rem' },
                        }}
                    >
                        {icon}
                    </Box>
                )}
                <Box>
                    <Typography variant="h5" fontWeight={700} lineHeight={1.2}>
                        {title}
                    </Typography>
                    {subtitle && (
                        <Typography variant="body2" color="text.secondary" mt={0.25}>
                            {subtitle}
                        </Typography>
                    )}
                </Box>
            </Box>
            {actions && <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>{actions}</Box>}
        </Box>
    );
}

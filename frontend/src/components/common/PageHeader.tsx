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
 * Follows modern design patterns with improved spacing and typography.
 */
export default function PageHeader({ title, subtitle, icon, actions }: PageHeaderProps) {
    return (
        <Box
            sx={{
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                mb: 4,
                flexWrap: "wrap",
                gap: 2,
            }}
        >
            <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
                {icon && (
                    <Box
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                            color: "primary.main",
                            flexShrink: 0,
                            '& .MuiSvgIcon-root': {
                                fontSize: '2rem',
                                transition: 'all 0.3s ease',
                            },
                        }}
                    >
                        {icon}
                    </Box>
                )}
                <Box>
                    <Typography
                        variant="h4"
                        fontWeight={800}
                        lineHeight={1.2}
                        sx={{
                            letterSpacing: '-0.01em',
                        }}
                    >
                        {title}
                    </Typography>
                    {subtitle && (
                        <Typography
                            variant="body2"
                            color="text.secondary"
                            mt={0.5}
                            sx={{
                                fontWeight: 500,
                            }}
                        >
                            {subtitle}
                        </Typography>
                    )}
                </Box>
            </Box>
            {actions && (
                <Box sx={{
                    display: "flex",
                    alignItems: "center",
                    gap: 1.5,
                    ml: 'auto',
                }}>
                    {actions}
                </Box>
            )}
        </Box>
    );
}

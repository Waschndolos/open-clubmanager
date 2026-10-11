import { Box, Typography } from "@mui/material";
import { alpha, useTheme } from "@mui/material/styles";
import CheckRoundedIcon from "@mui/icons-material/CheckRounded";
import { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import BrandMark from "../common/BrandMark";
import ThemeToggle from "../header/ThemeToggle";
import loginBg from "../../assets/login_bg.jpeg";

type AuthLayoutProps = {
    title: string;
    subtitle?: ReactNode;
    /** Optional element above the title, e.g. a step indicator. */
    eyebrow?: ReactNode;
    children: ReactNode;
    /** Width of the form column content. Setup needs a little more room. */
    formWidth?: number;
};

/**
 * Split-screen frame for login, password reset and first-time setup:
 * a photo panel that says what the app is for, and a calm form column.
 * Below the md breakpoint the photo panel is dropped and the form stands alone.
 */
export default function AuthLayout({ title, subtitle, eyebrow, children, formWidth = 400 }: AuthLayoutProps) {
    const { t } = useTranslation();
    const theme = useTheme();
    const points = t("auth.brand.points", { returnObjects: true }) as string[];

    return (
        <Box
            sx={{
                minHeight: "100vh",
                display: "grid",
                gridTemplateColumns: { xs: "1fr", md: "minmax(380px, 5fr) 6fr" },
                bgcolor: "background.default",
            }}
        >
            <Box
                component="aside"
                sx={{
                    display: { xs: "none", md: "flex" },
                    position: "relative",
                    flexDirection: "column",
                    justifyContent: "space-between",
                    p: { md: 5, lg: 6 },
                    color: "#FFFFFF",
                    overflow: "hidden",
                    backgroundColor: "#143D30",
                    backgroundImage: `linear-gradient(180deg, ${alpha("#0E2A21", 0.55)} 0%, ${alpha("#0E2A21", 0.2)} 40%, ${alpha("#0B211A", 0.92)} 100%), url(${loginBg})`,
                    backgroundSize: "cover",
                    backgroundPosition: "center",
                }}
            >
                <BrandMark inverted size={36} />

                <Box sx={{ maxWidth: 440 }}>
                    <Typography
                        component="p"
                        sx={{
                            fontFamily: theme.typography.h1.fontFamily,
                            fontWeight: 700,
                            fontSize: { md: "2.5rem", lg: "3rem" },
                            lineHeight: 1.05,
                            letterSpacing: "-0.03em",
                            mb: 3,
                        }}
                    >
                        {t("auth.brand.tagline")}
                    </Typography>
                    {Array.isArray(points) && (
                        <Box component="ul" sx={{ listStyle: "none", p: 0, m: 0, display: "grid", gap: 1.25 }}>
                            {points.map((point) => (
                                <Box component="li" key={point} sx={{ display: "flex", gap: 1.25, alignItems: "flex-start" }}>
                                    <CheckRoundedIcon sx={{ fontSize: 20, mt: "1px", color: theme.custom.accent }} />
                                    <Typography sx={{ color: alpha("#FFFFFF", 0.86), fontSize: "0.975rem" }}>{point}</Typography>
                                </Box>
                            ))}
                        </Box>
                    )}
                </Box>
            </Box>

            <Box
                component="main"
                sx={{
                    display: "flex",
                    flexDirection: "column",
                    minHeight: "100vh",
                    px: { xs: 2.5, sm: 4 },
                    py: { xs: 2.5, sm: 3 },
                }}
            >
                <Box sx={{ display: "flex", alignItems: "center", justifyContent: { xs: "space-between", md: "flex-end" } }}>
                    <Box sx={{ display: { xs: "block", md: "none" } }}>
                        <BrandMark size={30} />
                    </Box>
                    <ThemeToggle />
                </Box>

                <Box sx={{ flexGrow: 1, display: "flex", alignItems: "center", justifyContent: "center", py: { xs: 4, md: 6 } }}>
                    <Box sx={{ width: "100%", maxWidth: formWidth }}>
                        {eyebrow && <Box sx={{ mb: 2.5 }}>{eyebrow}</Box>}
                        <Typography variant="h4" component="h1" sx={{ mb: subtitle ? 1 : 3.5 }}>
                            {title}
                        </Typography>
                        {subtitle && (
                            <Typography color="text.secondary" sx={{ mb: 3.5, maxWidth: "60ch" }}>
                                {subtitle}
                            </Typography>
                        )}
                        {children}
                    </Box>
                </Box>
            </Box>
        </Box>
    );
}

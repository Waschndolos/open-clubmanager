import { Box, Typography } from "@mui/material";
import { alpha, useTheme } from "@mui/material/styles";
import CheckRoundedIcon from "@mui/icons-material/CheckRounded";
import { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import BrandMark from "../common/BrandMark";
import ThemeToggle from "../header/ThemeToggle";

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
 * a brand panel that says what the app is for, and a calm form column.
 * The panel is pure CSS (glowing gradients and a grid) so no image has to
 * be shipped. Below the md breakpoint the panel is dropped and the form stands alone.
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
                    isolation: "isolate",
                    backgroundColor: "#0A1F18",
                    backgroundImage: `radial-gradient(120% 80% at 0% 0%, ${alpha("#2F7A60", 0.55)} 0%, transparent 60%), linear-gradient(160deg, #0F2E24 0%, #0A1F18 55%, #07150F 100%)`,
                    "@keyframes authDrift": {
                        "0%": { transform: "translate3d(0, 0, 0) scale(1)" },
                        "50%": { transform: "translate3d(-6%, 5%, 0) scale(1.12)" },
                        "100%": { transform: "translate3d(0, 0, 0) scale(1)" },
                    },
                    "& > .auth-fx": { position: "absolute", pointerEvents: "none", zIndex: -1 },
                    "@media (prefers-reduced-motion: reduce)": {
                        "& .auth-anim": { animation: "none !important" },
                    },
                }}
            >
                {/* Ember glow */}
                <Box
                    className="auth-fx auth-anim"
                    aria-hidden
                    sx={{
                        width: "110%",
                        aspectRatio: "1",
                        right: "-35%",
                        bottom: "-45%",
                        borderRadius: "50%",
                        background: `radial-gradient(circle at 50% 50%, ${alpha("#FFC247", 1)} 0%, ${alpha("#F0731F", 0.85)} 25%, ${alpha("#C2410C", 0.45)} 50%, transparent 72%)`,
                        filter: "blur(18px)",
                        animation: "authDrift 16s ease-in-out infinite",
                    }}
                />
                <Box
                    className="auth-fx auth-anim"
                    aria-hidden
                    sx={{
                        width: "60%",
                        aspectRatio: "1",
                        left: "-25%",
                        top: "30%",
                        borderRadius: "50%",
                        background: `radial-gradient(circle, ${alpha("#5BC49A", 0.4)} 0%, transparent 65%)`,
                        filter: "blur(40px)",
                        animation: "authDrift 22s ease-in-out infinite reverse",
                    }}
                />
                {/* Fine grid, fading out towards the bottom */}
                <Box
                    className="auth-fx"
                    aria-hidden
                    sx={{
                        inset: 0,
                        backgroundImage: `linear-gradient(${alpha("#FFFFFF", 0.06)} 1px, transparent 1px), linear-gradient(90deg, ${alpha("#FFFFFF", 0.06)} 1px, transparent 1px)`,
                        backgroundSize: "44px 44px",
                        maskImage: "linear-gradient(180deg, #000 0%, transparent 75%)",
                        WebkitMaskImage: "linear-gradient(180deg, #000 0%, transparent 75%)",
                    }}
                />
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

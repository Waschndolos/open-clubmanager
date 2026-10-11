import { Box, Typography } from "@mui/material";
import { alpha, useTheme } from "@mui/material/styles";
import CheckRoundedIcon from "@mui/icons-material/CheckRounded";
import { ReactNode, useRef } from "react";
import { useTranslation } from "react-i18next";
import BrandMark from "../common/BrandMark";
import ThemeToggle from "../header/ThemeToggle";
import { tokens } from "../../theme/createAppTheme";
import { driftA, driftB, fadeIn, gridPan, pulseRing, reveal, slideInLeft } from "./authAnimations";

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
 * The panel is pure CSS (drifting orbs, a moving grid and a cursor spotlight)
 * so no image has to be shipped. Below the md breakpoint the panel is dropped and the form stands alone.
 */
export default function AuthLayout({ title, subtitle, eyebrow, children, formWidth = 400 }: AuthLayoutProps) {
    const { t } = useTranslation();
    const theme = useTheme();
    const points = t("auth.brand.points", { returnObjects: true }) as string[];
    const asideRef = useRef<HTMLElement>(null);
    const glow = tokens.dark;

    // Cursor spotlight: write CSS variables directly, no React re-renders on mousemove.
    const handlePointerMove = (e: React.PointerEvent<HTMLElement>) => {
        const el = asideRef.current;
        if (!el) return;
        const rect = el.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        el.style.setProperty("--mx", `${x}px`);
        el.style.setProperty("--my", `${y}px`);
        el.style.setProperty("--px", `${(x / rect.width - 0.5) * 2}`);
        el.style.setProperty("--py", `${(y / rect.height - 0.5) * 2}`);
    };

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
                ref={asideRef}
                onPointerMove={handlePointerMove}
                sx={{
                    "--mx": "50%",
                    "--my": "50%",
                    "--px": "0",
                    "--py": "0",
                    display: { xs: "none", md: "flex" },
                    position: "relative",
                    flexDirection: "column",
                    justifyContent: "space-between",
                    p: { md: 5, lg: 6 },
                    color: "#FFFFFF",
                    overflow: "hidden",
                    isolation: "isolate",
                    backgroundColor: "#143D30",
                    animation: `${fadeIn} 0.9s ease both`,
                    // Deep club-green gradient base
                    "&::after": {
                        content: '""',
                        position: "absolute",
                        inset: 0,
                        zIndex: -3,
                        backgroundImage: `linear-gradient(160deg, #143D30 0%, #0E2A21 55%, #0B211A 100%)`,
                    },
                }}
            >
                {/* Drifting colour orbs (club green + pennant amber) */}
                <Box aria-hidden sx={{ position: "absolute", inset: 0, zIndex: -2, pointerEvents: "none", mixBlendMode: "screen" }}>
                    <Box
                        sx={{
                            position: "absolute", top: "-10%", left: "-15%", width: "75%", aspectRatio: "1",
                            borderRadius: "50%", filter: "blur(70px)", opacity: 0.45,
                            background: `radial-gradient(circle, ${glow.primary} 0%, transparent 68%)`,
                            animation: `${driftA} 22s ease-in-out infinite`,
                        }}
                    />
                    <Box
                        sx={{
                            position: "absolute", bottom: "-18%", right: "-20%", width: "70%", aspectRatio: "1",
                            borderRadius: "50%", filter: "blur(80px)", opacity: 0.32,
                            background: `radial-gradient(circle, ${glow.accent} 0%, transparent 68%)`,
                            animation: `${driftB} 26s ease-in-out infinite`,
                        }}
                    />
                </Box>

                {/* Moving grid, faded out towards the edges */}
                <Box
                    aria-hidden
                    sx={{
                        position: "absolute", inset: 0, zIndex: -1, pointerEvents: "none", opacity: 0.5,
                        backgroundImage: `linear-gradient(${alpha("#FFFFFF", 0.07)} 1px, transparent 1px), linear-gradient(90deg, ${alpha("#FFFFFF", 0.07)} 1px, transparent 1px)`,
                        backgroundSize: "48px 48px",
                        animation: `${gridPan} 14s linear infinite`,
                        maskImage: "radial-gradient(ellipse 75% 65% at 50% 45%, #000 20%, transparent 100%)",
                        WebkitMaskImage: "radial-gradient(ellipse 75% 65% at 50% 45%, #000 20%, transparent 100%)",
                    }}
                />

                {/* Cursor spotlight */}
                <Box
                    aria-hidden
                    sx={{
                        position: "absolute", inset: 0, zIndex: -1, pointerEvents: "none",
                        background: `radial-gradient(420px circle at var(--mx) var(--my), ${alpha(glow.primaryLight, 0.2)}, transparent 70%)`,
                        transition: "background 0.15s linear",
                    }}
                />

                <Box sx={{ animation: reveal(0.15, slideInLeft), width: "fit-content" }}>
                    <BrandMark inverted animated size={36} />
                </Box>

                <Box
                    sx={{
                        maxWidth: 440,
                        transform: "translate3d(calc(var(--px) * -8px), calc(var(--py) * -6px), 0)",
                        transition: "transform 0.5s cubic-bezier(0.22, 1, 0.36, 1)",
                    }}
                >
                    <Typography
                        component="p"
                        sx={{
                            fontFamily: theme.typography.h1.fontFamily,
                            fontWeight: 700,
                            fontSize: { md: "2.5rem", lg: "3rem" },
                            lineHeight: 1.05,
                            letterSpacing: "-0.03em",
                            mb: 3,
                            animation: reveal(0.3, undefined, 1),
                        }}
                    >
                        {t("auth.brand.tagline")}
                    </Typography>
                    {Array.isArray(points) && (
                        <Box component="ul" sx={{ listStyle: "none", p: 0, m: 0, display: "grid", gap: 1.25 }}>
                            {points.map((point, i) => (
                                <Box
                                    component="li"
                                    key={point}
                                    sx={{ display: "flex", gap: 1.25, alignItems: "flex-start", animation: reveal(0.55 + i * 0.14, slideInLeft) }}
                                >
                                    <Box sx={{ position: "relative", display: "grid", placeItems: "center", mt: "1px", flexShrink: 0 }}>
                                        <Box
                                            aria-hidden
                                            sx={{
                                                position: "absolute", width: 20, height: 20, borderRadius: "50%",
                                                border: `1.5px solid ${theme.custom.accent}`,
                                                animation: `${pulseRing} 3.2s ease-out ${1.2 + i * 0.6}s infinite`,
                                            }}
                                        />
                                        <CheckRoundedIcon sx={{ fontSize: 20, color: theme.custom.accent }} />
                                    </Box>
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
                <Box sx={{ display: "flex", alignItems: "center", justifyContent: { xs: "space-between", md: "flex-end" }, animation: reveal(0.2, fadeIn, 0.8) }}>
                    <Box sx={{ display: { xs: "block", md: "none" } }}>
                        <BrandMark size={30} />
                    </Box>
                    <ThemeToggle />
                </Box>

                <Box sx={{ flexGrow: 1, display: "flex", alignItems: "center", justifyContent: "center", py: { xs: 4, md: 6 } }}>
                    <Box sx={{ width: "100%", maxWidth: formWidth }}>
                        {eyebrow && <Box sx={{ mb: 2.5, animation: reveal(0.2) }}>{eyebrow}</Box>}
                        <Typography variant="h4" component="h1" sx={{ mb: subtitle ? 1 : 3.5, animation: reveal(0.25) }}>
                            {title}
                        </Typography>
                        {subtitle && (
                            <Typography color="text.secondary" sx={{ mb: 3.5, maxWidth: "60ch", animation: reveal(0.35) }}>
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

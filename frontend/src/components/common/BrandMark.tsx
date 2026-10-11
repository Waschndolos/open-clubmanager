import { Box, Typography } from "@mui/material";
import { keyframes, useTheme } from "@mui/material/styles";

const drawRing = keyframes`
    from { stroke-dashoffset: 100; }
    to { stroke-dashoffset: 0; }
`;
const popDot = keyframes`
    0% { transform: scale(0); opacity: 0; }
    70% { transform: scale(1.35); opacity: 1; }
    100% { transform: scale(1); opacity: 1; }
`;

type BrandMarkProps = {
    /** Hide the wordmark and show only the badge. */
    compact?: boolean;
    /** Render light-on-dark, e.g. on top of a photo. */
    inverted?: boolean;
    size?: number;
    /** Draw the ring and pop the dot in on mount. */
    animated?: boolean;
};

/** Badge plus wordmark: an open "C" for club, with the accent dot sitting in the opening. */
export default function BrandMark({ compact = false, inverted = false, size = 32, animated = false }: BrandMarkProps) {
    const theme = useTheme();
    const badge = inverted ? "#FFFFFF" : theme.palette.primary.main;
    const ring = inverted ? theme.palette.primary.dark : theme.palette.primary.contrastText;

    return (
        <Box sx={{ display: "flex", alignItems: "center", gap: 1.25, minWidth: 0 }}>
            <Box
                component="svg"
                viewBox="0 0 32 32"
                aria-hidden
                sx={{ width: size, height: size, flexShrink: 0, display: "block" }}
            >
                <rect width="32" height="32" rx="8" fill={badge} />
                <path
                    d="M22.51 21.46 A8.5 8.5 0 1 1 22.51 10.54"
                    pathLength={100}
                    fill="none"
                    stroke={ring}
                    strokeWidth={3.4}
                    strokeLinecap="round"
                    strokeDasharray={100}
                    style={animated ? { animation: `${drawRing} 1.1s cubic-bezier(0.22, 1, 0.36, 1) 0.3s both` } : undefined}
                />
                <circle
                    cx={24.6}
                    cy={16}
                    r={2.3}
                    fill={theme.custom.accent}
                    style={animated ? { transformBox: "fill-box", transformOrigin: "center", animation: `${popDot} 0.5s ease-out 1.2s both` } : undefined}
                />
            </Box>
            {!compact && (
                <Typography
                    component="span"
                    noWrap
                    sx={{
                        fontFamily: theme.typography.h1.fontFamily,
                        fontWeight: 700,
                        fontSize: size * 0.53,
                        letterSpacing: "-0.02em",
                        color: inverted ? "#FFFFFF" : "text.primary",
                        lineHeight: 1,
                    }}
                >
                    Open ClubManager
                </Typography>
            )}
        </Box>
    );
}

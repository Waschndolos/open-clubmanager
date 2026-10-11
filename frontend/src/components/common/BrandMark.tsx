import { Box, Typography } from "@mui/material";
import { useTheme } from "@mui/material/styles";

type BrandMarkProps = {
    /** Hide the wordmark and show only the pennant badge. */
    compact?: boolean;
    /** Render light-on-dark, e.g. on top of a photo. */
    inverted?: boolean;
    size?: number;
};

/** Pennant badge plus wordmark. The pennant is the club's flag on the clubhouse. */
export default function BrandMark({ compact = false, inverted = false, size = 32 }: BrandMarkProps) {
    const theme = useTheme();
    const badge = inverted ? "#FFFFFF" : theme.palette.primary.main;
    const pole = inverted ? theme.palette.primary.dark : theme.palette.primary.contrastText;

    return (
        <Box sx={{ display: "flex", alignItems: "center", gap: 1.25, minWidth: 0 }}>
            <Box
                component="svg"
                viewBox="0 0 32 32"
                aria-hidden
                sx={{ width: size, height: size, flexShrink: 0, display: "block" }}
            >
                <rect width="32" height="32" rx="8" fill={badge} />
                <rect x="9" y="7" width="2.2" height="18" rx="1.1" fill={pole} />
                <path d="M11.2 8.2 L24 12.6 L11.2 17 Z" fill={theme.custom.accent} />
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

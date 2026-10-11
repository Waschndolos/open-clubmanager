import { alpha, createTheme, ThemeOptions } from '@mui/material/styles';

/**
 * Design tokens for Open ClubManager.
 *
 * Pine green is the club colour, pennant amber is the single accent and is
 * used sparingly (highlights, attention). Surfaces are separated by hairline
 * borders rather than shadows; radii follow hierarchy (surfaces > controls).
 */
export const tokens = {
    light: {
        primary: '#1E5A46',
        primaryLight: '#2F7A60',
        primaryDark: '#143D30',
        accent: '#C98A12',
        canvas: '#F4F6F3',
        surface: '#FFFFFF',
        sunken: '#EDF0EC',
        ink: '#15201B',
        inkMuted: '#5C6B64',
        line: '#E1E6E0',
    },
    dark: {
        primary: '#5BC49A',
        primaryLight: '#7FD4B1',
        primaryDark: '#3E9E78',
        accent: '#E3A72F',
        canvas: '#0F1513',
        surface: '#161E1B',
        sunken: '#121916',
        ink: '#E6EEE9',
        inkMuted: '#91A39A',
        line: '#24302B',
    },
} as const;

export const fontDisplay = '"Bricolage Grotesque Variable", "Figtree Variable", system-ui, sans-serif';
export const fontBody = '"Figtree Variable", system-ui, -apple-system, "Segoe UI", sans-serif';

export function createAppTheme(mode: 'light' | 'dark') {
    const c = tokens[mode];
    const isDark = mode === 'dark';

    const options: ThemeOptions = {
        palette: {
            mode,
            primary: {
                main: c.primary,
                light: c.primaryLight,
                dark: c.primaryDark,
                contrastText: isDark ? '#0C1712' : '#FFFFFF',
            },
            secondary: {
                main: c.accent,
                contrastText: '#1B1404',
            },
            info: { main: isDark ? '#6AB7E8' : '#2B6CB0' },
            success: { main: isDark ? '#5BC49A' : '#2E7D57' },
            warning: { main: isDark ? '#E3A72F' : '#B7791F' },
            error: { main: isDark ? '#F08A7E' : '#C0392B' },
            background: {
                default: c.canvas,
                paper: c.surface,
            },
            text: {
                primary: c.ink,
                secondary: c.inkMuted,
            },
            divider: c.line,
            action: {
                hover: alpha(c.primary, isDark ? 0.1 : 0.06),
                selected: alpha(c.primary, isDark ? 0.18 : 0.1),
            },
        },
        typography: {
            fontFamily: fontBody,
            h1: { fontFamily: fontDisplay, fontWeight: 700, letterSpacing: '-0.025em' },
            h2: { fontFamily: fontDisplay, fontWeight: 700, letterSpacing: '-0.02em' },
            h3: { fontFamily: fontDisplay, fontWeight: 700, letterSpacing: '-0.02em' },
            h4: { fontFamily: fontDisplay, fontWeight: 700, letterSpacing: '-0.015em', fontSize: '1.875rem' },
            h5: { fontFamily: fontDisplay, fontWeight: 650, letterSpacing: '-0.01em' },
            h6: { fontFamily: fontDisplay, fontWeight: 650, letterSpacing: '-0.005em', fontSize: '1.0625rem' },
            subtitle1: { fontWeight: 600 },
            subtitle2: { fontWeight: 600 },
            body1: { lineHeight: 1.55 },
            body2: { lineHeight: 1.5 },
            button: { fontWeight: 600, textTransform: 'none', letterSpacing: 0 },
            overline: { textTransform: 'none', letterSpacing: 0, fontWeight: 600 },
        },
        shape: { borderRadius: 10 },
        custom: {
            border: `1px solid ${c.line}`,
            boxShadow: isDark ? '0 1px 2px rgba(0,0,0,0.4)' : '0 1px 2px rgba(21,32,27,0.05)',
            accent: c.accent,
            sunken: c.sunken,
        },
        components: {
            MuiCssBaseline: {
                styleOverrides: {
                    body: { backgroundColor: c.canvas, color: c.ink },
                    '::selection': { backgroundColor: alpha(c.primary, 0.25) },
                    ':focus-visible': { outline: `2px solid ${c.primary}`, outlineOffset: 2 },
                },
            },
            MuiAppBar: {
                defaultProps: { elevation: 0, color: 'inherit' },
                styleOverrides: {
                    root: {
                        backgroundColor: alpha(c.canvas, 0.85),
                        backdropFilter: 'saturate(1.4) blur(10px)',
                        borderBottom: `1px solid ${c.line}`,
                        color: c.ink,
                    },
                },
            },
            MuiPaper: {
                defaultProps: { elevation: 0 },
                styleOverrides: {
                    root: { backgroundImage: 'none' },
                    rounded: { borderRadius: 14 },
                    outlined: { borderColor: c.line },
                    elevation0: { border: `1px solid ${c.line}` },
                },
            },
            MuiCard: {
                defaultProps: { elevation: 0 },
                styleOverrides: {
                    root: { border: `1px solid ${c.line}`, borderRadius: 14 },
                },
            },
            MuiButton: {
                defaultProps: { disableElevation: true },
                styleOverrides: {
                    root: { borderRadius: 8, paddingInline: 16, transition: 'background-color .15s, border-color .15s, color .15s' },
                    sizeLarge: { paddingBlock: 10, fontSize: '0.975rem' },
                    sizeSmall: { borderRadius: 7, paddingInline: 12 },
                    outlined: { borderColor: c.line, color: c.ink, '&:hover': { borderColor: alpha(c.primary, 0.5), backgroundColor: alpha(c.primary, 0.05) } },
                    containedPrimary: { '&:hover': { backgroundColor: isDark ? c.primaryLight : c.primaryLight } },
                },
            },
            MuiIconButton: {
                styleOverrides: {
                    root: { borderRadius: 8 },
                    sizeSmall: { padding: 7 },
                },
            },
            MuiOutlinedInput: {
                styleOverrides: {
                    root: {
                        backgroundColor: isDark ? c.sunken : c.surface,
                        borderRadius: 8,
                        '& .MuiOutlinedInput-notchedOutline': { borderColor: isDark ? '#2C3934' : '#D3DAD3' },
                        '&:hover .MuiOutlinedInput-notchedOutline': { borderColor: alpha(c.primary, 0.6) },
                        '&.Mui-focused .MuiOutlinedInput-notchedOutline': { borderColor: c.primary, borderWidth: 2 },
                    },
                },
            },
            MuiInputLabel: {
                styleOverrides: { root: { color: c.inkMuted } },
            },
            MuiListItemButton: {
                styleOverrides: {
                    root: {
                        borderRadius: 8,
                        '&.Mui-selected': {
                            backgroundColor: alpha(c.primary, isDark ? 0.16 : 0.09),
                            color: c.primary,
                            '& .MuiListItemIcon-root': { color: c.primary },
                        },
                        '&.Mui-selected:hover': { backgroundColor: alpha(c.primary, isDark ? 0.22 : 0.13) },
                    },
                },
            },
            MuiDialog: {
                styleOverrides: {
                    paper: {
                        border: `1px solid ${c.line}`,
                        borderRadius: 16,
                        boxShadow: isDark ? '0 24px 64px rgba(0,0,0,0.5)' : '0 24px 64px rgba(21,32,27,0.16)',
                    },
                },
            },
            MuiMenu: {
                styleOverrides: {
                    paper: {
                        border: `1px solid ${c.line}`,
                        borderRadius: 12,
                        boxShadow: isDark ? '0 12px 32px rgba(0,0,0,0.45)' : '0 12px 32px rgba(21,32,27,0.12)',
                    },
                },
            },
            MuiPopover: {
                styleOverrides: {
                    paper: {
                        border: `1px solid ${c.line}`,
                        borderRadius: 12,
                        boxShadow: isDark ? '0 12px 32px rgba(0,0,0,0.45)' : '0 12px 32px rgba(21,32,27,0.12)',
                    },
                },
            },
            MuiTooltip: {
                styleOverrides: {
                    tooltip: {
                        backgroundColor: isDark ? c.ink : '#1B2621',
                        color: isDark ? c.canvas : '#F4F6F3',
                        borderRadius: 6,
                        fontSize: '0.75rem',
                        fontWeight: 500,
                        padding: '5px 9px',
                    },
                    arrow: { color: isDark ? c.ink : '#1B2621' },
                },
            },
            MuiAlert: {
                styleOverrides: { root: { borderRadius: 10 } },
            },
            MuiChip: {
                styleOverrides: { root: { fontWeight: 600 } },
            },
        },
    };

    return createTheme(options);
}

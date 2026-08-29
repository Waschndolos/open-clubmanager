import { createTheme } from '@mui/material/styles';

const lightTheme = createTheme({
    palette: {
        mode: 'light',
        primary: {
            main: '#00D4AA',
            light: '#33E0BF',
            dark: '#00AA89',
            contrastText: '#FFFFFF',
        },
        secondary: {
            main: '#203436',
            light: '#4A5E62',
            dark: '#162122',
            contrastText: '#F5FFFC',
        },
        info: {
            main: '#0EA5E9',
        },
        success: {
            main: '#10B981',
        },
        warning: {
            main: '#F59E0B',
        },
        error: {
            main: '#EF4444',
        },
        background: {
            default: '#F1F5F4',
            paper: '#FFFFFF',
        },
        text: {
            primary: '#1A2E33',
            secondary: '#5E7A78',
        },
        divider: 'rgba(32, 52, 54, 0.12)',
        action: {
            hover: 'rgba(0, 212, 170, 0.08)',
            selected: 'rgba(0, 212, 170, 0.14)',
        },
    },
    typography: {
        fontFamily: '"Inter", "Space Grotesk", "Poppins", sans-serif',
        h1: { fontWeight: 800, letterSpacing: '-0.03em' },
        h2: { fontWeight: 800, letterSpacing: '-0.02em' },
        h3: { fontWeight: 700, letterSpacing: '-0.02em' },
        h4: { fontWeight: 700, letterSpacing: '-0.01em' },
        h5: { fontWeight: 700, letterSpacing: '-0.01em' },
        h6: { fontWeight: 650, letterSpacing: '-0.01em' },
        button: { fontWeight: 600, textTransform: 'none', letterSpacing: '0' },
        body1: { fontWeight: 400, letterSpacing: '0' },
        body2: { fontWeight: 400, letterSpacing: '0' },
        subtitle1: { fontWeight: 500 },
        subtitle2: { fontWeight: 500 },
    },
    shape: {
        borderRadius: 12,
    },
    custom: {
        border: '1px solid rgba(32, 52, 54, 0.12)',
        boxShadow: '0 4px 20px rgba(32, 52, 54, 0.08)',
    },
    components: {
        MuiCssBaseline: {
            styleOverrides: {
                body: {
                    backgroundColor: '#F1F5F4',
                    color: '#1A2E33',
                },
            },
        },
        MuiAppBar: {
            styleOverrides: {
                root: {
                    boxShadow: '0 1px 3px rgba(32, 52, 54, 0.08)',
                    backgroundColor: 'rgba(255, 255, 255, 0.92)',
                    backdropFilter: 'blur(12px)',
                    color: '#1A2E33',
                    borderBottom: '1px solid rgba(32, 52, 54, 0.08)',
                },
            },
        },
        MuiPaper: {
            styleOverrides: {
                root: {
                    backgroundColor: '#FFFFFF',
                    border: '1px solid rgba(32, 52, 54, 0.08)',
                    boxShadow: '0 4px 20px rgba(32, 52, 54, 0.06)',
                    borderRadius: 16,
                },
            },
        },
        MuiCard: {
            styleOverrides: {
                root: {
                    backgroundColor: '#FFFFFF',
                    border: '1px solid rgba(32, 52, 54, 0.08)',
                    boxShadow: '0 4px 20px rgba(32, 52, 54, 0.06)',
                    borderRadius: 16,
                },
            },
        },
        MuiButton: {
            styleOverrides: {
                root: {
                    borderRadius: 10,
                    textTransform: 'none',
                    fontWeight: 600,
                    letterSpacing: '0',
                    transition: 'all 0.2s cubic-bezier(0.4, 0, 0.2, 1)',
                    fontFamily: '"Inter", sans-serif',
                    boxShadow: 'none',
                },
                contained: {
                    boxShadow: '0 4px 14px rgba(0, 212, 170, 0.25)',
                    '&:hover': {
                        boxShadow: '0 6px 20px rgba(0, 212, 170, 0.35)',
                        transform: 'translateY(-1px)',
                    },
                    '&:active': {
                        transform: 'translateY(0)',
                    },
                    '&:disabled': {
                        backgroundColor: 'rgba(0, 212, 170, 0.35)',
                        color: 'rgba(255, 255, 255, 0.8)',
                    },
                },
                containedPrimary: {
                    background: 'linear-gradient(135deg, #00D4AA 0%, #00B894 100%)',
                },
                outlined: {
                    borderColor: 'rgba(32, 52, 54, 0.2)',
                    color: '#1A2E33',
                    '&:hover': {
                        backgroundColor: 'rgba(0, 212, 170, 0.06)',
                        borderColor: 'rgba(0, 212, 170, 0.5)',
                    },
                },
                text: {
                    color: '#00A88A',
                    '&:hover': {
                        backgroundColor: 'rgba(0, 212, 170, 0.08)',
                    },
                },
                sizeSmall: {
                    borderRadius: 8,
                    padding: '6px 14px',
                },
            },
        },
        MuiIconButton: {
            styleOverrides: {
                root: {
                    borderRadius: 10,
                    transition: 'all 0.2s ease',
                    '&:hover': {
                        backgroundColor: 'rgba(0, 212, 170, 0.1)',
                    },
                },
                sizeSmall: {
                    padding: 6,
                },
            },
        },
        MuiOutlinedInput: {
            styleOverrides: {
                root: {
                    backgroundColor: '#FFFFFF',
                    borderRadius: 10,
                    '& .MuiOutlinedInput-notchedOutline': {
                        borderColor: 'rgba(32, 52, 54, 0.16)',
                    },
                    '&:hover .MuiOutlinedInput-notchedOutline': {
                        borderColor: 'rgba(0, 212, 170, 0.6)',
                    },
                    '&.Mui-focused .MuiOutlinedInput-notchedOutline': {
                        borderColor: '#00D4AA',
                        borderWidth: 2,
                    },
                },
            },
        },
        MuiInputLabel: {
            styleOverrides: {
                root: {
                    color: '#5E7A78',
                },
            },
        },
        MuiListItemButton: {
            styleOverrides: {
                root: {
                    borderRadius: 10,
                    '&.Mui-selected': {
                        backgroundColor: 'rgba(0, 212, 170, 0.12)',
                        color: '#008F73',
                        '& .MuiListItemIcon-root': {
                            color: '#008F73',
                        },
                    },
                    '&.Mui-selected:hover': {
                        backgroundColor: 'rgba(0, 212, 170, 0.18)',
                    },
                },
            },
        },
        MuiDialog: {
            styleOverrides: {
                paper: {
                    background: '#FFFFFF',
                    backdropFilter: 'blur(12px)',
                    border: '1px solid rgba(32, 52, 54, 0.08)',
                    boxShadow: '0 20px 60px rgba(32, 52, 54, 0.12)',
                    borderRadius: 20,
                },
            },
        },
        MuiMenu: {
            styleOverrides: {
                paper: {
                    background: '#FFFFFF',
                    backdropFilter: 'blur(12px)',
                    border: '1px solid rgba(32, 52, 54, 0.08)',
                    borderRadius: 12,
                    boxShadow: '0 10px 40px rgba(32, 52, 54, 0.1)',
                },
            },
        },
        MuiTooltip: {
            styleOverrides: {
                tooltip: {
                    backgroundColor: '#203436',
                    borderRadius: 8,
                    fontSize: '0.75rem',
                    fontWeight: 500,
                    padding: '6px 10px',
                },
            },
        },
        MuiAlert: {
            styleOverrides: {
                root: {
                    borderRadius: 10,
                },
                outlined: {
                    borderWidth: 1,
                },
            },
        },
    },
});

export default lightTheme;

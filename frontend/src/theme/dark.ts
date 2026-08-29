import { createTheme } from '@mui/material/styles';

const darkTheme = createTheme({
    palette: {
        mode: 'dark',
        primary: {
            main: '#00D4AA',
            light: '#33E0BF',
            dark: '#00AA89',
            contrastText: '#FFFFFF',
        },
        secondary: {
            main: '#203436',
            light: '#2A3D40',
            dark: '#162122',
            contrastText: '#F5FFFC',
        },
        info: {
            main: '#38BDF8',
        },
        success: {
            main: '#34D399',
        },
        warning: {
            main: '#FBBF24',
        },
        error: {
            main: '#F87171',
        },
        background: {
            default: '#0F1516',
            paper: '#162122',
        },
        text: {
            primary: '#E8F4F1',
            secondary: '#8FA8A5',
        },
        divider: 'rgba(255, 255, 255, 0.08)',
        action: {
            hover: 'rgba(0, 212, 170, 0.1)',
            selected: 'rgba(0, 212, 170, 0.16)',
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
        border: '1px solid rgba(255, 255, 255, 0.08)',
        boxShadow: '0 8px 24px rgba(0, 0, 0, 0.3)',
    },
    components: {
        MuiCssBaseline: {
            styleOverrides: {
                body: {
                    backgroundColor: '#0F1516',
                    color: '#E8F4F1',
                },
            },
        },
        MuiAppBar: {
            styleOverrides: {
                root: {
                    boxShadow: '0 1px 3px rgba(0, 0, 0, 0.2)',
                    backgroundColor: 'rgba(22, 33, 34, 0.92)',
                    backdropFilter: 'blur(12px)',
                    color: '#E8F4F1',
                    borderBottom: '1px solid rgba(255, 255, 255, 0.06)',
                },
            },
        },
        MuiPaper: {
            styleOverrides: {
                root: {
                    backgroundColor: '#162122',
                    color: '#E8F4F1',
                    border: '1px solid rgba(255, 255, 255, 0.06)',
                    boxShadow: '0 8px 24px rgba(0, 0, 0, 0.25)',
                    borderRadius: 16,
                },
            },
        },
        MuiCard: {
            styleOverrides: {
                root: {
                    backgroundColor: '#162122',
                    border: '1px solid rgba(255, 255, 255, 0.06)',
                    boxShadow: '0 8px 24px rgba(0, 0, 0, 0.25)',
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
                        backgroundColor: 'rgba(0, 212, 170, 0.25)',
                        color: 'rgba(255, 255, 255, 0.6)',
                    },
                },
                containedPrimary: {
                    background: 'linear-gradient(135deg, #00D4AA 0%, #00B894 100%)',
                },
                outlined: {
                    borderColor: 'rgba(255, 255, 255, 0.15)',
                    color: '#E8F4F1',
                    '&:hover': {
                        backgroundColor: 'rgba(0, 212, 170, 0.08)',
                        borderColor: 'rgba(0, 212, 170, 0.5)',
                    },
                },
                text: {
                    color: '#33E0BF',
                    '&:hover': {
                        backgroundColor: 'rgba(0, 212, 170, 0.1)',
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
                        backgroundColor: 'rgba(0, 212, 170, 0.12)',
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
                    backgroundColor: '#0F1516',
                    borderRadius: 10,
                    '& .MuiOutlinedInput-notchedOutline': {
                        borderColor: 'rgba(255, 255, 255, 0.1)',
                    },
                    '&:hover .MuiOutlinedInput-notchedOutline': {
                        borderColor: 'rgba(0, 212, 170, 0.5)',
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
                    color: '#8FA8A5',
                },
            },
        },
        MuiListItemButton: {
            styleOverrides: {
                root: {
                    borderRadius: 10,
                    '&.Mui-selected': {
                        backgroundColor: 'rgba(0, 212, 170, 0.14)',
                        color: '#33E0BF',
                        '& .MuiListItemIcon-root': {
                            color: '#33E0BF',
                        },
                    },
                    '&.Mui-selected:hover': {
                        backgroundColor: 'rgba(0, 212, 170, 0.2)',
                    },
                },
            },
        },
        MuiDialog: {
            styleOverrides: {
                paper: {
                    background: '#162122',
                    backdropFilter: 'blur(12px)',
                    border: '1px solid rgba(255, 255, 255, 0.06)',
                    boxShadow: '0 20px 60px rgba(0, 0, 0, 0.35)',
                    borderRadius: 20,
                },
            },
        },
        MuiMenu: {
            styleOverrides: {
                paper: {
                    background: '#162122',
                    backdropFilter: 'blur(12px)',
                    border: '1px solid rgba(255, 255, 255, 0.06)',
                    borderRadius: 12,
                    boxShadow: '0 10px 40px rgba(0, 0, 0, 0.3)',
                },
            },
        },
        MuiTooltip: {
            styleOverrides: {
                tooltip: {
                    backgroundColor: '#E8F4F1',
                    color: '#162122',
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

export default darkTheme;

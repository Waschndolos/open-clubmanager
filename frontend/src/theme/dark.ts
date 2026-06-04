import { createTheme } from '@mui/material/styles';

const darkTheme = createTheme({
    palette: {
        mode: 'dark',
        primary: {
            main: '#00FFC2',
            light: '#6FFFE0',
            dark: '#00C89A',
            contrastText: '#07251E',
        },
        secondary: {
            main: '#2D3436',
            light: '#4B5558',
            dark: '#1D2224',
            contrastText: '#EFFFF9',
        },
        info: {
            main: '#22D3EE',
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
            default: '#111516',
            paper: '#1A2022',
        },
        text: {
            primary: '#E8FFFA',
            secondary: '#B5D1CB',
        },
        divider: 'rgba(181, 209, 203, 0.24)',
        action: {
            hover: 'rgba(0, 255, 194, 0.12)',
            selected: 'rgba(0, 255, 194, 0.20)',
        },
    },
    typography: {
        fontFamily: 'Inter, Roboto, Helvetica, Arial, sans-serif',
        h1: { fontWeight: 750, letterSpacing: '-0.02em' },
        h2: { fontWeight: 700, letterSpacing: '-0.02em' },
        h3: { fontWeight: 700, letterSpacing: '-0.01em' },
        h4: { fontWeight: 680 },
        h5: { fontWeight: 650 },
        h6: { fontWeight: 620 },
        button: { fontWeight: 650, textTransform: 'none' },
    },
    shape: {
        borderRadius: 14,
    },
    custom: {
        border: '1px solid rgba(181, 209, 203, 0.24)',
        boxShadow: '0 12px 30px rgba(0, 0, 0, 0.42)',
    },
    components: {
        MuiCssBaseline: {
            styleOverrides: {
                body: {
                    backgroundColor: '#111516',
                    color: '#E8FFFA',
                },
            },
        },
        MuiAppBar: {
            styleOverrides: {
                root: {
                    boxShadow: '0 1px 0 rgba(181, 209, 203, 0.20)',
                    backgroundColor: 'rgba(26, 32, 34, 0.82)',
                    backdropFilter: 'blur(10px)',
                    color: '#E8FFFA',
                },
            },
        },
        MuiPaper: {
            styleOverrides: {
                root: {
                    backgroundColor: 'rgba(26, 32, 34, 0.8)',
                    color: '#E8FFFA',
                    border: '1px solid rgba(181, 209, 203, 0.15)',
                    boxShadow: '0 8px 24px rgba(0, 0, 0, 0.2)',
                    backdropFilter: 'blur(8px)',
                },
            },
        },
        MuiCard: {
            styleOverrides: {
                root: {
                    border: '1px solid rgba(181, 209, 203, 0.22)',
                    boxShadow: '0 12px 30px rgba(0, 0, 0, 0.38)',
                },
            },
        },
        MuiButton: {
            styleOverrides: {
                root: {
                    borderRadius: 10,
                    textTransform: 'none',
                    fontWeight: 650,
                    letterSpacing: '0.01em',
                    transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
                    position: 'relative',
                    overflow: 'hidden',
                },
                contained: {
                    boxShadow: '0 8px 20px rgba(0, 255, 194, 0.25)',
                    '&:hover': {
                        boxShadow: '0 12px 28px rgba(0, 255, 194, 0.35)',
                        transform: 'translateY(-2px)',
                    },
                    '&:active': {
                        transform: 'translateY(0)',
                    }
                },
                outlined: {
                    borderColor: 'rgba(0, 255, 194, 0.4)',
                    '&:hover': {
                        backgroundColor: 'rgba(0, 255, 194, 0.08)',
                        borderColor: 'rgba(0, 255, 194, 0.6)',
                    }
                },
                text: {
                    '&:hover': {
                        backgroundColor: 'rgba(0, 255, 194, 0.08)',
                    }
                }
            },
        },
        MuiOutlinedInput: {
            styleOverrides: {
                root: {
                    backgroundColor: 'rgba(22, 27, 29, 0.66)',
                    '& .MuiOutlinedInput-notchedOutline': {
                        borderColor: 'rgba(181, 209, 203, 0.30)',
                    },
                    '&:hover .MuiOutlinedInput-notchedOutline': {
                        borderColor: 'rgba(181, 209, 203, 0.38)',
                    },
                    '&.Mui-focused .MuiOutlinedInput-notchedOutline': {
                        borderColor: 'rgba(181, 209, 203, 0.48)',
                        borderWidth: 1,
                    },
                },
            },
        },
        MuiDialog: {
            styleOverrides: {
                paper: {
                    background: 'rgba(26, 32, 34, 0.95)',
                    backdropFilter: 'blur(12px)',
                    border: '1px solid rgba(0, 255, 194, 0.12)',
                    boxShadow: '0 20px 60px rgba(0, 0, 0, 0.3)',
                }
            }
        },
        MuiMenu: {
            styleOverrides: {
                paper: {
                    background: 'rgba(26, 32, 34, 0.95)',
                    backdropFilter: 'blur(12px)',
                    border: '1px solid rgba(0, 255, 194, 0.12)',
                }
            }
        },
    },
});

export default darkTheme;

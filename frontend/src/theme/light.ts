import { createTheme } from '@mui/material/styles';

const lightTheme = createTheme({
    palette: {
        mode: 'light',
        primary: {
            main: '#00FFC2',
            light: '#00FFC2',
            dark: '#00FFC2',
            contrastText: '#203436',
        },
        secondary: {
            main: '#203436',
            light: '#4A5E62',
            dark: '#162122',
            contrastText: '#F5FFFC',
        },
        info: {
            main: '#00FFC2',
        },
        success: {
            main: '#00FFC2',
        },
        warning: {
            main: '#D97706',
        },
        error: {
            main: '#DC2626',
        },
        background: {
            default: '#F8FFFE',
            paper: '#FFFFFF',
        },
        text: {
            primary: '#203436',
            secondary: '#6B8A87',
        },
        divider: '#203436',
        action: {
            hover: '#F0F5F5',
            selected: '#E8F5F3',
        },
    },
    typography: {
        fontFamily: '"Space Grotesk", "Poppins", "Inter", sans-serif',
        h1: { fontWeight: 800, letterSpacing: '-0.02em' },
        h2: { fontWeight: 800, letterSpacing: '-0.02em' },
        h3: { fontWeight: 750, letterSpacing: '-0.01em' },
        h4: { fontWeight: 720 },
        h5: { fontWeight: 700 },
        h6: { fontWeight: 650 },
        button: { fontWeight: 700, textTransform: 'none', letterSpacing: '0.01em' },
        body1: { fontWeight: 500 },
        body2: { fontWeight: 500 },
        subtitle1: { fontWeight: 600 },
        subtitle2: { fontWeight: 600 },
    },
    shape: {
        borderRadius: 14,
    },
    custom: {
        border: '1px solid #203436',
        boxShadow: '0 6px 20px rgba(32, 52, 54, 0.1)',
    },
    components: {
        MuiCssBaseline: {
            styleOverrides: {
                body: {
                    backgroundColor: '#F8FFFE',
                    color: '#203436',
                },
            },
        },
        MuiAppBar: {
            styleOverrides: {
                root: {
                    boxShadow: '0 2px 8px rgba(32, 52, 54, 0.1)',
                    backgroundColor: '#FFFFFF',
                    backdropFilter: 'blur(20px)',
                    color: '#203436',
                    borderBottom: '1px solid #203436',
                },
            },
        },
        MuiPaper: {
            styleOverrides: {
                root: {
                    backgroundColor: '#FFFFFF',
                    border: '1px solid #203436',
                    boxShadow: '0 6px 20px rgba(32, 52, 54, 0.08)',
                    backdropFilter: 'blur(8px)',
                },
            },
        },
        MuiCard: {
            styleOverrides: {
                root: {
                    backgroundColor: '#FFFFFF',
                    border: '1px solid #203436',
                    boxShadow: '0 6px 20px rgba(32, 52, 54, 0.08)',
                    backdropFilter: 'blur(8px)',
                },
            },
        },
        MuiButton: {
            styleOverrides: {
                root: {
                    borderRadius: 10,
                    textTransform: 'none',
                    fontWeight: 700,
                    letterSpacing: '0.01em',
                    transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
                    position: 'relative',
                    overflow: 'hidden',
                    fontFamily: '"Space Grotesk", sans-serif',
                },
                contained: {
                    boxShadow: '0 6px 18px rgba(0, 255, 194, 0.24)',
                    '&:hover': {
                        boxShadow: '0 10px 26px rgba(0, 255, 194, 0.32)',
                        transform: 'translateY(-2px)',
                    },
                    '&:active': {
                        transform: 'translateY(0)',
                    }
                },
                outlined: {
                    borderColor: '#00FFC2',
                    color: '#00FFC2',
                    '&:hover': {
                        backgroundColor: '#F0F5F5',
                        borderColor: '#00FFC2',
                    }
                },
                text: {
                    color: '#00FFC2',
                    '&:hover': {
                        backgroundColor: '#F0F5F5',
                    }
                }
            },
        },
        MuiOutlinedInput: {
            styleOverrides: {
                root: {
                    backgroundColor: '#F8FFFE',
                    '& .MuiOutlinedInput-notchedOutline': {
                        borderColor: '#203436',
                    },
                    '&:hover .MuiOutlinedInput-notchedOutline': {
                        borderColor: '#00FFC2',
                    },
                    '&.Mui-focused .MuiOutlinedInput-notchedOutline': {
                        borderColor: '#00FFC2',
                        borderWidth: 2,
                    },
                },
            },
        },
        MuiDialog: {
            styleOverrides: {
                paper: {
                    background: '#FFFFFF',
                    backdropFilter: 'blur(12px)',
                    border: '1px solid #203436',
                    boxShadow: '0 20px 60px rgba(32, 52, 54, 0.12)',
                }
            }
        },
        MuiMenu: {
            styleOverrides: {
                paper: {
                    background: '#FFFFFF',
                    backdropFilter: 'blur(12px)',
                    border: '1px solid #203436',
                }
            }
        },
    },
});

export default lightTheme;

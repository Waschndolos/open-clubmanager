import { createTheme } from '@mui/material/styles';

const darkTheme = createTheme({
    palette: {
        mode: 'dark',
        primary: {
            main: '#00FFC2',
            light: '#00FFC2',
            dark: '#00FFC2',
            contrastText: '#203436',
        },
        secondary: {
            main: '#203436',
            light: '#162122',
            dark: '#0F1516',
            contrastText: '#00FFC2',
        },
        info: {
            main: '#00FFC2',
        },
        success: {
            main: '#00FFC2',
        },
        warning: {
            main: '#FFB800',
        },
        error: {
            main: '#FF6B6B',
        },
        background: {
            default: '#0F1516',
            paper: '#162122',
        },
        text: {
            primary: '#E8F4F1',
            secondary: '#A8C5C1',
        },
        divider: '#203436',
        action: {
            hover: '#203436',
            selected: '#162122',
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
                    boxShadow: '0 2px 8px rgba(0, 0, 0, 0.25)',
                    backgroundColor: '#162122',
                    backdropFilter: 'blur(20px)',
                    color: '#E8F4F1',
                    borderBottom: '1px solid #203436',
                },
            },
        },
        MuiPaper: {
            styleOverrides: {
                root: {
                    backgroundColor: '#162122',
                    color: '#E8F4F1',
                    border: '1px solid #203436',
                    boxShadow: '0 8px 24px rgba(0, 0, 0, 0.25)',
                    backdropFilter: 'blur(12px)',
                },
            },
        },
        MuiCard: {
            styleOverrides: {
                root: {
                    backgroundColor: '#162122',
                    border: '1px solid #203436',
                    boxShadow: '0 8px 24px rgba(0, 0, 0, 0.25)',
                    backdropFilter: 'blur(12px)',
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
                    borderColor: '#00FFC2',
                    color: '#00FFC2',
                    '&:hover': {
                        backgroundColor: '#203436',
                        borderColor: '#00FFC2',
                    }
                },
                text: {
                    color: '#00FFC2',
                    '&:hover': {
                        backgroundColor: '#203436',
                    }
                }
            },
        },
        MuiOutlinedInput: {
            styleOverrides: {
                root: {
                    backgroundColor: '#0F1516',
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
                    background: '#162122',
                    backdropFilter: 'blur(12px)',
                    border: '1px solid #203436',
                    boxShadow: '0 20px 60px rgba(0, 0, 0, 0.3)',
                }
            }
        },
        MuiMenu: {
            styleOverrides: {
                paper: {
                    background: '#162122',
                    backdropFilter: 'blur(12px)',
                    border: '1px solid #203436',
                }
            }
        },
    },
});

export default darkTheme;

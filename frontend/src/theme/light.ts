import { createTheme } from '@mui/material/styles';

const lightTheme = createTheme({
    palette: {
        mode: 'light',
        primary: {
            main: '#00C89A',
            light: '#45FFD2',
            dark: '#009C78',
            contrastText: '#0D1B18',
        },
        secondary: {
            main: '#2D3436',
            light: '#4A5356',
            dark: '#1F2527',
            contrastText: '#F5FFFC',
        },
        info: {
            main: '#0EA5A3',
        },
        success: {
            main: '#00B88D',
        },
        warning: {
            main: '#D97706',
        },
        error: {
            main: '#DC2626',
        },
        background: {
            default: '#F3F8F7',
            paper: '#FFFFFF',
        },
        text: {
            primary: '#2D3436',
            secondary: '#4F5A5D',
        },
        divider: 'rgba(45, 52, 54, 0.18)',
        action: {
            hover: 'rgba(0, 255, 194, 0.12)',
            selected: 'rgba(0, 255, 194, 0.18)',
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
        border: '1px solid rgba(45, 52, 54, 0.16)',
        boxShadow: '0 10px 30px rgba(45, 52, 54, 0.10)',
    },
    components: {
        MuiCssBaseline: {
            styleOverrides: {
                body: {
                    backgroundColor: '#F3F8F7',
                    color: '#2D3436',
                },
            },
        },
        MuiAppBar: {
            styleOverrides: {
                root: {
                    boxShadow: '0 1px 0 rgba(45, 52, 54, 0.14)',
                    backgroundColor: 'rgba(255, 255, 255, 0.88)',
                    backdropFilter: 'blur(10px)',
                    color: '#2D3436',
                },
            },
        },
        MuiPaper: {
            styleOverrides: {
                root: {
                    border: '1px solid rgba(45, 52, 54, 0.14)',
                    boxShadow: '0 8px 24px rgba(45, 52, 54, 0.10)',
                },
            },
        },
        MuiCard: {
            styleOverrides: {
                root: {
                    border: '1px solid rgba(45, 52, 54, 0.14)',
                    boxShadow: '0 10px 28px rgba(45, 52, 54, 0.10)',
                },
            },
        },
        MuiButton: {
            styleOverrides: {
                root: {
                    borderRadius: 10,
                    textTransform: 'none',
                },
                contained: {
                    boxShadow: '0 8px 20px rgba(0, 200, 154, 0.28)',
                },
            },
        },
        MuiOutlinedInput: {
            styleOverrides: {
                root: {
                    backgroundColor: 'rgba(255, 255, 255, 0.78)',
                    '& .MuiOutlinedInput-notchedOutline': {
                        borderColor: 'rgba(45, 52, 54, 0.22)',
                    },
                    '&:hover .MuiOutlinedInput-notchedOutline': {
                        borderColor: 'rgba(45, 52, 54, 0.30)',
                    },
                    '&.Mui-focused .MuiOutlinedInput-notchedOutline': {
                        borderColor: 'rgba(45, 52, 54, 0.38)',
                        borderWidth: 1,
                    },
                },
            },
        },
    },
});

export default lightTheme;

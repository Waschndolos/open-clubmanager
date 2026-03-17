import { createTheme } from '@mui/material/styles';

const lightTheme = createTheme({
    palette: {
        mode: 'light',
        primary: {
            main: '#4F6AF5',
            light: '#7B91F8',
            dark: '#3451D1',
            contrastText: '#ffffff',
        },
        secondary: {
            main: '#7C3AED',
        },
        background: {
            default: '#F8FAFC',
            paper: '#ffffff',
        },
    },
    shape: {
        borderRadius: 12,
    },
    custom: {
        border: '1px solid #E2E8F0',
        boxShadow: '0 1px 4px rgba(0,0,0,0.06)',
    },
    components: {
        MuiAppBar: {
            styleOverrides: {
                root: {
                    boxShadow: '0 1px 0 #E2E8F0',
                    backgroundColor: '#ffffff',
                    color: '#0F172A',
                },
            },
        },
        MuiButton: {
            styleOverrides: {
                containedPrimary: {
                    boxShadow: '0 1px 2px rgba(79,106,245,0.25)',
                    '&:hover': {
                        boxShadow: '0 2px 6px rgba(79,106,245,0.35)',
                    },
                },
            },
        },
    },
})

export default lightTheme
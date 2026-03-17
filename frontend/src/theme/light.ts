import { createTheme } from '@mui/material/styles';

const lightTheme = createTheme({
    palette: {
        mode: 'light',
        primary: {
            main: '#819A91',
        },
        secondary: {
            main: '#f50057',
        },
        background: {
            default: '#f0f2f5',
            paper: '#ffffff',
        },
    },
    shape: {
        borderRadius: 14,
    },
    custom: {
        border: '1px solid #e0e7ef',
        boxShadow: '0 1px 4px rgba(0,0,0,0.06)',
    },
    components: {
        MuiAppBar: {
            styleOverrides: {
                root: {
                    boxShadow: '0 1px 3px rgba(0,0,0,0.12)',
                },
            },
        },
    },
})

export default lightTheme
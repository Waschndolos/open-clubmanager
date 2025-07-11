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
            default: '#fff',
            paper: '#fff',
        },
    },
    shape: {
        borderRadius: 14,
    },
    custom: {
        border: '1px solid #e0e7ef',
        boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
    },
})

export default lightTheme
import { createTheme } from '@mui/material/styles'

const lightTheme = createTheme({
    palette: {
        mode: 'light',
        primary: {
            main: '#819A91',
        },
        secondary: {
            main: '#A7C1A8',
        },
        background: {
            default: '#f4f6f8',
        },
    },
})

export default lightTheme

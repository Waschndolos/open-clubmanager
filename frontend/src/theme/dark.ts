import { createTheme } from '@mui/material/styles'

const darkTheme = createTheme({
    palette: {
        mode: 'dark',
        primary: {
            main: '#819A91',
        },
        secondary: {
            main: '#f50057',
        },
        background: {
            default: '#1E1F22',
        },
        divider: "rgba(255, 255, 255, 0.12)"
    },
    shape: {
        borderRadius: 14,
    },
    components: {
        MuiPaper: {
            styleOverrides: {
                root: {
                    backgroundColor: '#2B2D30',
                    color: '#fff',
                    borderRadius: 14,
                    border: "1px solid rgba(255, 255, 255, 0.12)",
                    boxShadow: '0 1px 6px 0 rgba(255, 255, 255, 0.12)',
                    transition: 'background 0.2s',
                }
            }
        }
    },
    custom: {
        border: "1px solid rgba(255, 255, 255, 0.12)",
        boxShadow: '0 1px 6px 0 rgba(255, 255, 255, 0.12)'
    }
})

export default darkTheme
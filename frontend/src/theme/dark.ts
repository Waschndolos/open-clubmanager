import { createTheme } from '@mui/material/styles'

const darkTheme = createTheme({
    palette: {
        mode: 'dark',
        primary: {
            main: '#7C8EF8',
            light: '#A5B2FB',
            dark: '#4F6AF5',
            contrastText: '#ffffff',
        },
        secondary: {
            main: '#A78BFA',
        },
        background: {
            default: '#0F1117',
            paper: '#1A1D27',
        },
        divider: "rgba(255, 255, 255, 0.08)"
    },
    shape: {
        borderRadius: 12,
    },
    components: {
        MuiPaper: {
            styleOverrides: {
                root: {
                    backgroundColor: '#1A1D27',
                    color: '#E2E8F0',
                    borderRadius: 12,
                    border: "1px solid rgba(255, 255, 255, 0.08)",
                    boxShadow: 'none',
                    transition: 'background 0.2s',
                }
            }
        },
        MuiAppBar: {
            styleOverrides: {
                root: {
                    boxShadow: '0 1px 0 rgba(255,255,255,0.08)',
                    backgroundColor: '#1A1D27',
                    color: '#E2E8F0',
                },
            },
        },
    },
    custom: {
        border: "1px solid rgba(255, 255, 255, 0.08)",
        boxShadow: 'none'
    }
})

export default darkTheme
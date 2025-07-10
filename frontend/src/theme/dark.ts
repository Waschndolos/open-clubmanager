import { createTheme } from '@mui/material/styles'

const darkTheme = createTheme({
    palette: {
        mode: 'dark',
        primary: {
            main: '#8ecae6', // Accent blue
            contrastText: '#1a1a2e',
        },
        secondary: {
            main: '#ffb703', // Accent yellow
            contrastText: '#1a1a2e',
        },
        background: {
            default: '#181926', // Main background
            paper: '#23243a',  // Cards and menus
        },
        text: {
            primary: '#f4f6fb',
            secondary: '#b0b3c6',
            disabled: '#6c6f7e',
        },
        divider: '#31324b',
    },
    shape: {
        borderRadius: 14,
    },
    components: {
        MuiPaper: {
            styleOverrides: {
                root: {
                    backgroundImage: 'none',
                    boxShadow: '0 4px 24px 0 rgba(0,0,0,0.25)',
                },
            },
        },
        MuiAppBar: {
            styleOverrides: {
                colorPrimary: {
                    backgroundColor: '#23243a',
                    boxShadow: '0 2px 8px 0 rgba(0,0,0,0.18)',
                },
            },
        },
        MuiListItem: {
            styleOverrides: {
                root: {
                    borderRadius: 8,
                    marginBottom: 2,
                    transition: 'background 0.2s',
                    '&.Mui-selected, &.Mui-selected:hover': {
                        backgroundColor: '#23243a',
                        color: '#8ecae6',
                    },
                },
            },
        },
        MuiButton: {
            styleOverrides: {
                root: {
                    borderRadius: 8,
                    textTransform: 'none',
                    fontWeight: 500,
                },
            },
        },
    },
})

export default darkTheme

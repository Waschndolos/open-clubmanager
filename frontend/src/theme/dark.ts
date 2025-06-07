import { createTheme } from '@mui/material/styles'

const darkTheme = createTheme({
    palette: {
        mode: 'dark',
        primary: {
            main: '#90caf9',
        },
        background: {
            default: '#121212',
        },
    },
})

export default darkTheme

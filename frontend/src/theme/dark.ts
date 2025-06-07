import { createTheme } from '@mui/material/styles'

const darkTheme = createTheme({
    palette: {
        mode: 'dark',
        primary: {
            main: '#948979',
        },
        secondary: {
            main: '#DFD0B8',
        },
        background: {
            default: '#393E46',
        },
    },
})

export default darkTheme

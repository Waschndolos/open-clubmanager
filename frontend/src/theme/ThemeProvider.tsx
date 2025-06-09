import {PropsWithChildren, useMemo, useState} from 'react'
import {ThemeProvider as MuiThemeProvider, CssBaseline} from '@mui/material'
import {lightTheme, darkTheme} from '.'
import {AdapterDayjs} from "@mui/x-date-pickers/AdapterDayjs";
import {LocalizationProvider} from "@mui/x-date-pickers";

export default function ThemeProvider({children}: PropsWithChildren) {
    const [mode, setMode] = useState<'light' | 'dark'>('light')

    const theme = useMemo(() => (mode === 'light' ? lightTheme : darkTheme), [mode])

    const toggleTheme = () => setMode((prev) => (prev === 'light' ? 'dark' : 'light'))

    return (

        <MuiThemeProvider theme={theme}>
            <LocalizationProvider dateAdapter={AdapterDayjs}>
                <CssBaseline/>
                <div style={{position: 'fixed', top: 10, right: 10}}>
                    <button onClick={toggleTheme}>🌗</button>
                </div>
                {children}
            </LocalizationProvider>
        </MuiThemeProvider>

    )
}

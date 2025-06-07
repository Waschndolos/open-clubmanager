import { PropsWithChildren, useMemo, useState } from 'react'
import { ThemeProvider as MuiThemeProvider, CssBaseline } from '@mui/material'
import { lightTheme, darkTheme } from '.'

export default function ThemeProvider({ children }: PropsWithChildren) {
    const [mode, setMode] = useState<'light' | 'dark'>('light')

    const theme = useMemo(() => (mode === 'light' ? lightTheme : darkTheme), [mode])

    const toggleTheme = () => setMode((prev) => (prev === 'light' ? 'dark' : 'light'))

    return (
        <MuiThemeProvider theme={theme}>
            <CssBaseline />
            {/* 👇 Kontext oder Toggle später hier reinreichen */}
            <div style={{ position: 'fixed', top: 10, right: 10 }}>
                <button onClick={toggleTheme}>🌗</button>
            </div>
            {children}
        </MuiThemeProvider>
    )
}

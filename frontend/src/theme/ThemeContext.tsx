import { createContext, useContext, useState, useMemo, ReactNode } from 'react'
import { ThemeProvider as MuiThemeProvider, createTheme, CssBaseline } from '@mui/material'
import { lightTheme, darkTheme } from '.'

type ThemeMode = 'light' | 'dark'

interface ThemeContextType {
    mode: ThemeMode
    toggleTheme: () => void
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

export function useThemeContext() {
    const context = useContext(ThemeContext)
    if (!context) throw new Error('useThemeContext must be used within ThemeProvider')
    return context
}

export function ThemeProvider({ children }: { children: ReactNode }) {
    const [mode, setMode] = useState<ThemeMode>('light')

    const toggleTheme = () => setMode((prev) => (prev === 'light' ? 'dark' : 'light'))

    const theme = useMemo(() => createTheme(mode === 'light' ? lightTheme : darkTheme), [mode])

    return (
        <ThemeContext.Provider value={{ mode, toggleTheme }}>
            <MuiThemeProvider theme={theme}>
                <CssBaseline />
                {children}
            </MuiThemeProvider>
        </ThemeContext.Provider>
    )
}

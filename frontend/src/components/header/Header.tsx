import { AppBar, Toolbar, Typography, IconButton } from '@mui/material'
import { LightMode, DarkMode } from '@mui/icons-material'
import { useThemeContext } from '../../theme/ThemeContext'

export default function Header() {
    const { mode, toggleTheme } = useThemeContext()

    return (
        <AppBar position="static" color="primary">
            <Toolbar>
                <Typography variant="h6" sx={{ flexGrow: 1 }}>
                   Open ClubManager
                </Typography>
                <IconButton color="inherit" onClick={toggleTheme}>
                    {mode === 'light' ? <DarkMode /> : <LightMode />}
                </IconButton>
            </Toolbar>
        </AppBar>
    )
}

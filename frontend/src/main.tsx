import {createRoot} from 'react-dom/client'
import './index.css'
import {RouterProvider} from "react-router";
import {router} from "./routes/routes"
import '@fontsource/roboto/300.css';
import {CssBaseline} from "@mui/material";
import {ThemeProvider} from './theme/ThemeContext';

createRoot(document.getElementById('root')!).render(
    <ThemeProvider>
        <CssBaseline />
        <RouterProvider router={router} />
    </ThemeProvider>
)

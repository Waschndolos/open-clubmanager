import {createRoot} from 'react-dom/client'
import './index.css'
import {RouterProvider} from "react-router";
import {router} from "./routes/routes"
import '@fontsource/roboto/300.css';
import {CssBaseline} from "@mui/material";
import {ThemeProvider} from './theme/ThemeContext';
import './i18n';
import {NotificationProvider} from "./components/header/NotificationContext";
import 'ag-grid-community/styles/ag-theme-balham.css';
import {AuthProvider} from "./context/AuthContext";


createRoot(document.getElementById('root')!).render(
    <ThemeProvider>

        <NotificationProvider>

            <CssBaseline/>
            <AuthProvider>
                <RouterProvider router={router}/>
            </AuthProvider>
        </NotificationProvider>

    </ThemeProvider>
)

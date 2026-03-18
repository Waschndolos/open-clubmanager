import {RouterProvider} from "react-router";
import {router} from "./routes/routes";
import {ThemeProvider} from './theme/ThemeContext';
import {NotificationProvider} from "./context/NotificationContext";
import {AuthProvider} from "./context/AuthContext";
import './i18n';
import '@fontsource/roboto/300.css';

/**
 * Root application component that composes all application-level providers
 * and renders the router. This follows the standard React/Vite pattern of
 * separating bootstrapping (main.tsx) from the application tree (App.tsx).
 */
export default function App() {
    return (
        <ThemeProvider>
            <NotificationProvider>
                <AuthProvider>
                    <RouterProvider router={router}/>
                </AuthProvider>
            </NotificationProvider>
        </ThemeProvider>
    );
}

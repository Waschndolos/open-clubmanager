import React, {JSX} from "react";
import { Box, CircularProgress } from "@mui/material";
import { useAuth } from "../context/AuthContext";
import { Navigate, useLocation } from "react-router-dom";

/** Router state the login page reads to explain the redirect and return. */
export interface LoginRedirectState {
    from?: string;
    sessionExpired?: boolean;
}

export const RequireAuth: React.FC<{ children: JSX.Element }> = ({ children }) => {
    const { accessToken, initializing, sessionExpired } = useAuth();
    const location = useLocation();

    if (!accessToken) {
        // Wait for the silent refresh on app start before deciding.
        if (initializing) {
            return (
                <Box sx={{ display: "flex", justifyContent: "center", p: 6 }}>
                    <CircularProgress />
                </Box>
            );
        }
        const state: LoginRedirectState = {
            from: location.pathname + location.search + location.hash,
            sessionExpired,
        };
        return <Navigate to="/login" replace state={state} />;
    }

    return children;
};

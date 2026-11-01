import React, { createContext, useCallback, useContext, useEffect, useState, ReactNode } from "react";
import { AppRole } from "../api/types";
import {
    onAccessTokenChange,
    onSessionExpired,
    refreshSession,
    setAccessToken as setApiAccessToken,
} from "../api/api";
import { logout as logoutRequest } from "../api/authentication";

interface AuthContextType {
    accessToken: string | null;
    appRole: AppRole | null;
    email: string | null;
    /** True until the silent refresh on app start has finished. */
    initializing: boolean;
    /** True after a session ended because it could not be refreshed. */
    sessionExpired: boolean;
    setAccessToken: (token: string | null) => void;
    /** Ends the session on the server and locally; the caller navigates. */
    logout: () => void;
}

/** Decodes the payload section of a JWT without verifying the signature. */
function decodeJwtPayload(token: string): Record<string, unknown> | null {
    try {
        const base64 = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
        const json = atob(base64);
        return JSON.parse(json);
    } catch {
        return null;
    }
}

function extractRole(token: string | null): AppRole | null {
    if (!token) return null;
    const payload = decodeJwtPayload(token);
    if (!payload) return null;
    const role = payload["appRole"] as string;
    if (["ADMIN", "TREASURER", "SECRETARY", "READONLY", "TRAINER"].includes(role)) {
        return role as AppRole;
    }
    return null;
}

function extractEmail(token: string | null): string | null {
    if (!token) return null;
    const email = decodeJwtPayload(token)?.["email"];
    return typeof email === "string" ? email : null;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
    const [accessToken, setAccessTokenState] = useState<string | null>(null);
    const [initializing, setInitializing] = useState(true);
    const [sessionExpired, setSessionExpired] = useState(false);

    // The API client owns the token; it changes on login, on every refresh and
    // when a session ends, so mirror it here.
    useEffect(() => onAccessTokenChange((token) => {
        setAccessTokenState(token);
        if (token) setSessionExpired(false);
    }), []);

    useEffect(() => {
        onSessionExpired(() => setSessionExpired(true));
        return () => onSessionExpired(null);
    }, []);

    // A reload or a deep link starts without an access token: try the refresh
    // token before anything redirects to the login.
    useEffect(() => {
        let cancelled = false;
        refreshSession()
            .catch(() => undefined)
            .finally(() => {
                if (!cancelled) setInitializing(false);
            });
        return () => {
            cancelled = true;
        };
    }, []);

    const setAccessToken = useCallback((token: string | null) => setApiAccessToken(token), []);

    const logout = useCallback(() => {
        setSessionExpired(false);
        setApiAccessToken(null);
        void logoutRequest();
    }, []);

    return (
        <AuthContext.Provider value={{
            accessToken,
            appRole: extractRole(accessToken),
            email: extractEmail(accessToken),
            initializing,
            sessionExpired,
            setAccessToken,
            logout,
        }}>
            {children}
        </AuthContext.Provider>
    );
};

export const useAuth = (): AuthContextType => {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error("useAuth must be used within AuthProvider");
    }
    return context;
};

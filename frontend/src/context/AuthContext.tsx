import React, { createContext, useContext, useState, ReactNode } from "react";
import { AppRole } from "../api/types";

interface AuthContextType {
    accessToken: string | null;
    appRole: AppRole | null;
    setAccessToken: (token: string | null) => void;
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
    if (["ADMIN", "TREASURER", "SECRETARY", "READONLY"].includes(role)) {
        return role as AppRole;
    }
    return null;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
    const [accessToken, setAccessTokenState] = useState<string | null>(null);
    const [appRole, setAppRole] = useState<AppRole | null>(null);

    const setAccessToken = (token: string | null) => {
        setAccessTokenState(token);
        setAppRole(extractRole(token));
    };

    const logout = () => {
        setAccessTokenState(null);
        setAppRole(null);
        window.location.href = "/login";
    };

    return (
        <AuthContext.Provider value={{ accessToken, appRole, setAccessToken, logout }}>
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


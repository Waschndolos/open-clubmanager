import React, { JSX } from "react";
import { Navigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { AppRole } from "../api/types";

interface RequireRoleProps {
    roles: AppRole[];
    children: JSX.Element;
}

/**
 * Redirects to /dashboard when the authenticated user does not hold one of
 * the required roles. Must be used inside RequireAuth (which ensures that a
 * valid access token exists).
 */
export const RequireRole: React.FC<RequireRoleProps> = ({ roles, children }) => {
    const { appRole } = useAuth();

    if (!appRole || !roles.includes(appRole)) {
        return <Navigate to="/dashboard" replace />;
    }

    return children;
};

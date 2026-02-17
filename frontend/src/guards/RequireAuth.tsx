import React, {JSX} from "react";
import { useAuth } from "../context/AuthContext";
import { Navigate } from "react-router-dom";

export const RequireAuth: React.FC<{ children: JSX.Element }> = ({ children }) => {
    const { accessToken } = useAuth();

    if (!accessToken) {
        return <Navigate to="/login" replace />;
    }

    return children;
};

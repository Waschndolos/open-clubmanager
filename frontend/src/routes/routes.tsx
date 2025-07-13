import { createBrowserRouter, Navigate } from "react-router-dom";
import NotFound from "../pages/NotFound";
import MainLayout from "../layouts/MainLayout";
import Dashboard from "../pages/dashboard/Dashboard";
import Members from "../pages/members/Members";
import {Finance} from "../pages/finance/Finance";
import { Settings } from "../pages/settings/Settings";
import {EntitiesPage} from "../pages/entities/EntitiesPage";
import Login from "../pages/login/Login";
import { RequireAuth } from "../guards/RequireAuth";

export const router = createBrowserRouter([
    {
    path: "/login",
    element: <Login />,
  },
  {
        path: "/",
        element: <MainLayout />,
        children: [
            {
                index: true,
                element: <Navigate to="/dashboard" replace />,
            },
            {
        path: "dashboard",
        element: (
          <RequireAuth>
            <Dashboard />
          </RequireAuth>
        ),
            },
            {
        path: "members",
        element: (
          <RequireAuth>
            <Members />
          </RequireAuth>
        ),
            },
            {
        path: "finance",
        element: (
          <RequireAuth>
            <Finance />
          </RequireAuth>
        ),
            },
            {
        path: "settings",
        element: (
          <RequireAuth>
            <Settings />
          </RequireAuth>
        ),
            },
            {
        path: "entities",
        element: (
          <RequireAuth>
            <EntitiesPage />
          </RequireAuth>
        ),
            },
            {
                path: "*",
        element: <NotFound />,
      },
    ],
  },
]);

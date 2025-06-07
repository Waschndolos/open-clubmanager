import {createBrowserRouter} from "react-router";
import NotFound from "../pages/NotFound";
import MainLayout from "../layouts/MainLayout";
import Dashboard from "../components/dashboard/Dashboard";
import Members from "../components/members/Members";
import {Finance} from "../components/finance/Finance";
import { Settings } from "../components/settings/Settings";

export const router = createBrowserRouter([
    {
        path: "/",
        Component: MainLayout,
        children: [
            {
                path: "/dashboard",
                Component: Dashboard,
            },
            {
                path: "/members",
                Component: Members
            },
            {
                path: "/finance",
                Component: Finance
            },
            {
                path: "/settings",
                Component: Settings
            },
            {
                path: "*",
                Component: NotFound
            }
        ]
    }
])
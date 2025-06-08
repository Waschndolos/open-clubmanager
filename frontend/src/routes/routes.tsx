import {createBrowserRouter} from "react-router";
import NotFound from "../pages/NotFound";
import MainLayout from "../layouts/MainLayout";
import Dashboard from "../pages/dashboard/Dashboard";
import Members from "../pages/members/Members";
import {Finance} from "../pages/finance/Finance";
import { Settings } from "../pages/settings/Settings";
import {EntitiesPage} from "../pages/entities/EntitiesPage";

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
                path: "/entities",
                Component: EntitiesPage
            },
            {
                path: "*",
                Component: NotFound
            }
        ]
    }
])
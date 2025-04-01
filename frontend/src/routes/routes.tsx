import {createBrowserRouter} from "react-router";
import NotFound from "../pages/NotFound";
import Home from "../pages/Home";
import MainLayout from "../layouts/MainLayout";

export let router = createBrowserRouter([
    {
        path: "/",
        Component: MainLayout,
        children: [
            {
                index: true,
                Component: Home,
            },
            {
                path: "*",
                Component: NotFound
            }
        ]
    }
])
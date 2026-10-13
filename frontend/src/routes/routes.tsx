import { createBrowserRouter, Navigate } from "react-router-dom";
import NotFound from "../pages/NotFound";
import MainLayout from "../layouts/MainLayout";
import Dashboard from "../pages/dashboard/Dashboard";
import Members from "../pages/members/Members";
import {Finance} from "../pages/finance/Finance";
import { Settings } from "../pages/settings/Settings";
import {EntitiesPage} from "../pages/entities/EntitiesPage";
import Login from "../pages/login/Login";
import ForgotPassword from "../pages/login/ForgotPassword";
import ResetPassword from "../pages/login/ResetPassword";
import Setup from "../pages/setup/Setup";
import { RequireAuth } from "../guards/RequireAuth";
import { RequireRole } from "../guards/RequireRole";
import History from "../pages/history/History";
import Users from "../pages/users/Users";
import Inventory from "../pages/inventory/Inventory";
import Tasks from "../pages/tasks/Tasks";
import Events from "../pages/events/Events";
import Households from "../pages/households/Households";
import Honors from "../pages/honors/Honors";
import Qualifications from "../pages/qualifications/Qualifications";
import CollaborationSettings from "../pages/collaboration/CollaborationSettings";
import CreateClubWizard from "../pages/collaboration/CreateClubWizard";
import JoinClubWizard from "../pages/collaboration/JoinClubWizard";
import RecoverWizard from "../pages/collaboration/RecoverWizard";
import Conflicts from "../pages/collaboration/Conflicts";

export const router = createBrowserRouter([
    {
    path: "/login",
    element: <Login />,
  },
  {
    path: "/forgot-password",
    element: <ForgotPassword />,
  },
  {
    path: "/reset-password",
    element: <ResetPassword />,
  },
  {
    path: "/setup",
    element: <Setup />,
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
        path: "history",
        element: (
          <RequireAuth>
            <History />
          </RequireAuth>
        ),
            },
            {
        path: "inventory",
        element: (
          <RequireAuth>
            <Inventory />
          </RequireAuth>
        ),
            },
            {
        path: "honors",
        element: (
          <RequireAuth>
            <Honors />
          </RequireAuth>
        ),
            },
            {
        path: "qualifications",
        element: (
          <RequireAuth>
            <RequireRole roles={["ADMIN", "SECRETARY"]}>
              <Qualifications />
            </RequireRole>
          </RequireAuth>
        ),
            },
            {
        path: "tasks",
        element: (
          <RequireAuth>
            <Tasks />
          </RequireAuth>
        ),
            },
            {
        path: "events",
        element: (
          <RequireAuth>
            <Events />
          </RequireAuth>
        ),
            },
            {
        path: "households",
        element: (
          <RequireAuth>
            <Households />
          </RequireAuth>
        ),
            },
            {
        path: "users",
        element: (
          <RequireAuth>
            <RequireRole roles={["ADMIN"]}>
              <Users />
            </RequireRole>
          </RequireAuth>
        ),
            },
            {
        path: "collaboration",
        element: (
          <RequireAuth>
            <CollaborationSettings />
          </RequireAuth>
        ),
            },
            {
        path: "collaboration/create",
        element: (
          <RequireAuth>
            <CreateClubWizard />
          </RequireAuth>
        ),
            },
            {
        path: "collaboration/join",
        element: (
          <RequireAuth>
            <JoinClubWizard />
          </RequireAuth>
        ),
            },
            {
        path: "collaboration/recover",
        element: (
          <RequireAuth>
            <RecoverWizard />
          </RequireAuth>
        ),
            },
            {
        path: "collaboration/conflicts",
        element: (
          <RequireAuth>
            <Conflicts />
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

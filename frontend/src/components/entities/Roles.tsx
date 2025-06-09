import {EntityManager} from "./EntityManager";
import {useTranslation} from "react-i18next";
import {Group} from "../api/types";
import {createRole, deleteRole, fetchRoles, updateRole} from "../api/roles";

export function Roles() {

    const { t } = useTranslation();
    return (
        <EntityManager<Group>
            description={t("entities.roles.description")}
            fetchFn={fetchRoles}
            createFn={createRole}
            updateFn={updateRole}
            deleteFn={deleteRole}
            labels={{
                name: t("entities.roles.dialogs.name"),
                description: t("entities.roles.dialogs.description"),
                createButton:  t("entities.roles.dialogs.createButton"),
            }}
        />
    )
}
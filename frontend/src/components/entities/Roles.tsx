import {EntityManager} from "./EntityManager";
import {useTranslation} from "react-i18next";
import {Group} from "../api/types";
import {createRole, fetchRoles} from "../api/roles";

export function Roles() {

    const { t } = useTranslation();
    return (
        <EntityManager<Group>
            title={t("entities.roles.title")}
            fetchFn={fetchRoles}
            createFn={createRole}
            labels={{
                name: "Role name",
                description: "Beschreibung",
                createButton: "Neue Gruppe",
            }}
        />
    )
}
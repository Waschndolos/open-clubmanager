import {EntityManager} from "./EntityManager";
import {useTranslation} from "react-i18next";
import {Group} from "../api/types";
import {createGroup, fetchGroups} from "../api/groups";

export function Groups() {

    const { t } = useTranslation();
    return (
        <EntityManager<Group>
            title={t("entities.groups.title")}
            fetchFn={fetchGroups}
            createFn={createGroup}
            labels={{
                name: "Gruppenname",
                description: "Beschreibung",
                createButton: "Neue Gruppe",
            }}
        />
    )
}
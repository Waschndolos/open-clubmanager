import {EntityManager} from "./EntityManager";
import {useTranslation} from "react-i18next";
import {Group} from "../../api/types";
import {createGroup, deleteGroup, fetchGroups, updateGroup} from "../../api/groups";

export function Groups() {

    const { t } = useTranslation();
    return (
        <EntityManager<Group>
            description={t("entities.groups.description")}
            fetchFn={fetchGroups}
            createFn={createGroup}
            deleteFn={deleteGroup}
            updateFn={updateGroup}
            labels={{
                name: t("entities.groups.dialogs.name"),
                description: t("entities.groups.dialogs.description"),
                createButton:  t("entities.groups.dialogs.createButton"),
            }}
        />
    )
}
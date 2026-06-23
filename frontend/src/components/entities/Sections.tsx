import {EntityManager} from "./EntityManager";
import {useTranslation} from "react-i18next";
import {ClubSection} from "../../api/types";
import {assignSectionMembers, createSection, deleteSection, fetchSectionMembers, fetchSections, updateSection} from "../../api/sections";

export function Sections() {

    const { t } = useTranslation();
    return (
        <EntityManager<ClubSection>
            description={t("entities.sections.description")}
            fetchFn={fetchSections}
            createFn={createSection}
            updateFn={updateSection}
            deleteFn={deleteSection}
            fetchMembersFn={fetchSectionMembers}
            assignMembersFn={assignSectionMembers}
            labels={{
                name: t("entities.sections.dialogs.name"),
                description: t("entities.sections.dialogs.description"),
                createButton:  t("entities.sections.dialogs.createButton"),
            }}
        />
    )
}
import {EntityManager} from "./EntityManager";
import {useTranslation} from "react-i18next";
import {ClubSection} from "../api/types";
import {createSection, fetchSections} from "../api/sections";

export function Sections() {

    const { t } = useTranslation();
    return (
        <EntityManager<ClubSection>
            title={t("entities.sections.title")}
            fetchFn={fetchSections}
            createFn={createSection}
            labels={{
                name: "Section name",
                description: "Beschreibung",
                createButton: "Neue Gruppe",
            }}
        />
    )
}
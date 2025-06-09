import {Box, Tab, Tabs} from "@mui/material";
import {useTranslation} from "react-i18next";
import React, {useState} from "react";
import { Groups } from "../../components/entities/Groups";
import {Roles} from "../../components/entities/Roles";
import {Sections} from "../../components/entities/Sections";


function a11yProps(index: number) {
    return {
        id: `simple-tab-${index}`,
        'aria-controls': `simple-tabpanel-${index}`,
    };
}
interface TabPanelProps {
    children?: React.ReactNode;
    index: number;
    value: number;
}

function CustomTabPanel(props: TabPanelProps) {
    const { children, value, index, ...other } = props;

    return (
        <div
            role="tabpanel"
            hidden={value !== index}
            id={`simple-tabpanel-${index}`}
            aria-labelledby={`simple-tab-${index}`}
            {...other}
        >
            {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
        </div>
    );
}

export function EntitiesPage() {

    const {t} = useTranslation();
    const [value, setValue] = useState(0);

    const handleChange = (event: React.SyntheticEvent, newValue: number) => {
        setValue(newValue)
    }

    return (
        <Box sx={{display: 'flex', flexDirection: 'column', height: '100%', width: '100%'}}>
            <Tabs value={value} onChange={handleChange} variant={"fullWidth"}>
                <Tab label={t("entities.groups.title")} {...a11yProps(0)} />
                <Tab label={t("entities.roles.title")} {...a11yProps(1)} />
                <Tab label={t("entities.sections.title")} {...a11yProps(2)} />
            </Tabs>
            <CustomTabPanel value={value} index={0}>
                <Groups />
            </CustomTabPanel>
            <CustomTabPanel value={value} index={1}>
                <Roles />
            </CustomTabPanel>
            <CustomTabPanel value={value} index={2}>
                <Sections />
            </CustomTabPanel>
        </Box>


    )
}
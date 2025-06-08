import {Box, Checkbox, FormControlLabel, IconButton, Menu, MenuItem, Typography} from "@mui/material";
import {useTranslation} from "react-i18next";
import {AgGridReact} from "ag-grid-react";
import {useMemo, useRef, useState} from "react";
import {Member} from "./types";
import {AllCommunityModule, ModuleRegistry, themeMaterial} from "ag-grid-community";
import {ViewColumn} from "@mui/icons-material";

// Register only the community modules
ModuleRegistry.registerModules([AllCommunityModule]);

type MemberTableProps = {
    members: Member[];
};

export default function MemberTable({members}: MemberTableProps) {
    const {t} = useTranslation();
    const gridRef = useRef<AgGridReact>(null);
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const columnDefs = useMemo(() => {
        if (!members || members.length === 0) return [];

        const keys = Object.keys(members[0]) as (keyof Member)[];

        return keys.map((key, index) => ({
            field: key,
            headerName: t("members.table.header." + key),
            sortable: true,
            filter: false,
            resizable: true,
            flex: 1,
            hide: index >= 10 // show only first 10 initially
        }));
    }, [members, t]);

    const handleOpenMenu = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };
    const handleCloseMenu = () => {
        setAnchorEl(null);
    };
    const menuOpen = Boolean(anchorEl);

    const keys = useMemo(() => {
        if (!members || members.length === 0) return [];
        return Object.keys(members[0]) as (keyof Member)[];
    }, [members]);

    if (!members || members.length === 0) {
        return <Typography>No members found</Typography>;
    }

    return (
        <Box sx={{height: "calc(100vh - 300px)", width: "100%"}}>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={1}>
                <Typography variant="h5">{t("members.title")}</Typography>
                <IconButton onClick={handleOpenMenu}>
                    <ViewColumn/>
                </IconButton>
                <Menu anchorEl={anchorEl} open={menuOpen} onClose={handleCloseMenu} sx={{height: "50%"}}>
                    {keys.map((key) => (
                        <MenuItem key={key} dense disableGutters>
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        defaultChecked={
                                            columnDefs.find((c) => c.field === key)?.hide === false
                                        }
                                        onChange={(e) =>
                                            gridRef.current?.api.setColumnsVisible([key], e.target.checked)
                                        }
                                    />
                                }
                                label={t("members.table.header." + key)}
                            />
                        </MenuItem>
                    ))}
                </Menu>
            </Box>
            <div className="ag-theme-alpine" style={{height: "100%", width: "100%"}}>
                <AgGridReact
                    ref={gridRef}
                    rowData={members}
                    columnDefs={columnDefs}
                    theme={themeMaterial}
                    defaultColDef={{sortable: true, filter: true, resizable: true}}
                    pagination
                    paginationPageSize={20}
                />
            </div>


        </Box>
    );
}

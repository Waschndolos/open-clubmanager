import {Box, Checkbox, FormControlLabel, IconButton, Menu, MenuItem, Typography} from "@mui/material";
import {useTranslation} from "react-i18next";
import {AgGridReact} from "ag-grid-react";
import React, {useMemo, useRef, useState} from "react";
import {Member} from "../../components/api/types";
import {AllCommunityModule, ModuleRegistry, themeMaterial} from "ag-grid-community";
import {ViewColumn} from "@mui/icons-material";
import {EditMemberDialog} from "./EditMemberDialog";
import EditIcon from '@mui/icons-material/Edit';
import {updateMember} from "../../components/api/members";

ModuleRegistry.registerModules([AllCommunityModule]);

type MemberTableProps = {
    members: Member[];
    onMemberUpdated: (newMember: Member) => void;
};

export default function MemberTable({members, onMemberUpdated}: MemberTableProps) {
    const {t} = useTranslation();
    const gridRef = useRef<AgGridReact>(null);
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const [editingMember, setEditingMember] = useState<Member | null>(null);
    const [selectedMember, setSelectedMember] = useState<Member | null>(null);
    const keys = useMemo(() => {
        if (!members || members.length === 0) return [];
        return Object.keys(members[0]) as (keyof Member)[];
    }, [members]);

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

    const [columnVisibility, setColumnVisibility] = useState<Record<string, boolean>>(() => {
        const initial = {} as Record<string, boolean>;
        keys.forEach(key => {
            initial[key] = columnDefs.find(col => col.field === key)?.hide !== true;
        });
        return initial;
    });



    const handleSelectionChanged = () => {
        const selectedNodes = gridRef.current?.api.getSelectedNodes();
        const selectedData = selectedNodes?.[0]?.data;
        setSelectedMember(selectedData || null);
    };

    const handleOpenMenu = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };
    const handleEditMember = () => {
        setEditingMember(selectedMember)
    }

    const handleCloseMenu = () => {
        setAnchorEl(null);
    };

    const handleColumnToggle = (key: string) => (event: React.ChangeEvent<HTMLInputElement>) => {
        const visible = event.target.checked;
        setColumnVisibility(prev => ({ ...prev, [key]: visible }));
        gridRef.current?.api.setColumnsVisible([key], visible);
    };

    const handleSaveEdit = async (updated: Member) => {
        updateMember(updated).then((updatedMember: Member) => {
            onMemberUpdated(updatedMember);
            setEditingMember(null);
        })
    };

    const menuOpen = Boolean(anchorEl);


    if (!members || members.length === 0) {
        return <Typography>No members found</Typography>;
    }

    return (
        <Box sx={{height: "calc(100vh - 300px)", width: "100%"}}>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={1}>
                <Typography variant="body1">{t("members.table.description")}</Typography>

                <Box display="flex" justifyContent="end">
                    <IconButton onClick={handleEditMember} disabled={!selectedMember}>
                        <EditIcon/>
                    </IconButton>
                    <IconButton onClick={handleOpenMenu}>
                        <ViewColumn/>
                    </IconButton>
                </Box>

                <Menu anchorEl={anchorEl} open={menuOpen} onClose={handleCloseMenu} sx={{height: "50%"}}>
                    {keys.map((key) => (
                        <MenuItem key={key} dense disableGutters>
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        checked={columnVisibility[key]}
                                        onChange={handleColumnToggle(key)}
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
                    columnDefs={columnDefs.map((col) => ({
                        ...col,
                        hide: !columnVisibility[col.field ?? ""],
                    }))}
                    theme={themeMaterial}
                    defaultColDef={{sortable: true, filter: true, resizable: true}}
                    pagination
                    paginationPageSize={20}
                    rowSelection={"single"}
                    onSelectionChanged={handleSelectionChanged}
                />
            </div>
            {editingMember && (
                <EditMemberDialog
                    member={editingMember!}
                    onClose={() => setEditingMember(null)}
                    onSave={(update: Member) => {
                        handleSaveEdit(update);
                    }}
                />
            )}
        </Box>
    );

}

import {Box, Checkbox, FormControlLabel, IconButton, Menu, MenuItem, Typography} from "@mui/material";
import {useTranslation} from "react-i18next";
import {AgGridReact} from "ag-grid-react";
import React, {useMemo, useRef, useState} from "react";
import {Member} from "./types";
import {AllCommunityModule, ModuleRegistry, themeMaterial} from "ag-grid-community";
import {ViewColumn} from "@mui/icons-material";
import {EditMemberDialog} from "./EditMemberDialog";
import EditIcon from '@mui/icons-material/Edit';

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

    const handleSaveEdit = async (updated: Member) => {
        try {
            const response = await fetch(`http://localhost:3001/api/members/${updated.id}`, {
                method: "PUT",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(updated),
            });

            if (!response.ok) {
                throw new Error("Update failed");
            }
            const backendResponse = await response.json() as Member;
            onMemberUpdated(backendResponse);
            setEditingMember(null);
        } catch (error) {
            console.error("Fehler beim Speichern:", error);
        }
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

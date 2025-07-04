import {Box, Checkbox, FormControlLabel, IconButton, Menu, MenuItem, Tooltip, Typography} from "@mui/material";
import {useTranslation} from "react-i18next";
import {AgGridReact} from "ag-grid-react";
import React, {useEffect, useMemo, useRef, useState} from "react";
import {Member} from "../../components/api/types";
import {AllCommunityModule, ModuleRegistry, themeMaterial} from "ag-grid-community";
import {Delete, FileDownload, ViewColumn} from "@mui/icons-material";
import {EditMemberDialog} from "./EditMemberDialog";
import EditIcon from '@mui/icons-material/Edit';
import {deleteMember, updateMember} from "../../components/api/members";
import {useUserPreference} from "../../hooks/useUserPreference";
import {DateRenderer, DefaultRenderer, MemberContainingNamedArtifactRenderer} from "./renderer";
import {useThemeContext} from "../../theme/ThemeContext";
import {DeletingMemberDialog} from "./DeletingMemberDialog";
import {ExportMembersDialog} from "./ExportMembersDialog";

ModuleRegistry.registerModules([AllCommunityModule]);

type MemberTableProps = {
    members: Member[];
    onMemberUpdated: (newMember: Member) => void;
    onMemberDeleted: (deletedMember: Member) => void;
};

export default function MemberTable({members, onMemberUpdated, onMemberDeleted}: MemberTableProps) {
    const {t} = useTranslation();
    const gridRef = useRef<AgGridReact>(null);
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const [editingMember, setEditingMember] = useState<Member | null>(null);
    const [exportOpened, setExportOpened] = useState<boolean>(false);
    const [deletingMember, setDeletingMember] = useState<Member | null>(null);
    const [selectedMember, setSelectedMember] = useState<Member | null>(null);
    const {getPreference, setPreference} = useUserPreference();
    const keys = useMemo(() => {
        if (!members || members.length === 0) return [];
        return Object.keys(members[0]) as (keyof Member)[];
    }, [members]);
    const themeContext = useThemeContext();

    const columnDefs = useMemo(() => {
        if (!members || members.length === 0) return [];

        const keys = Object.keys(members[0]) as (keyof Member)[];

        const getCellRenderer = (key: "id" | "number" | "firstName" | "lastName" | "email" | "birthday" | "phone" | "phoneMobile" | "comment" | "entryDate" | "exitDate" | "street" | "postalCode" | "city" | "state" | "accountHolder" | "iban" | "bic" | "bankName" | "sepaMandateDate" | "roles" | "groups" | "sections") => {
            switch (key) {
                case "entryDate":
                case "exitDate":
                case "birthday":
                    return DateRenderer
                case "roles":
                case "groups":
                case "sections":
                    return MemberContainingNamedArtifactRenderer
                default:
                    return DefaultRenderer
            }
        };
        return keys.map((key, index) => ({
            field: key,
            headerName: t("members.table.header." + key),
            sortable: true,
            filter: false,
            resizable: true,
            cellRenderer: getCellRenderer(key),
            flex: 1,
            autoHeight: true,
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


    useEffect(() => {
        if (!keys.length) return;
        getPreference("memberTable.columns").then((stored) => {
            const fallback = Object.fromEntries(keys.map((k, i) => [k, i < 10])); // default: first 10 true
            setColumnVisibility(stored ?? fallback);
        });
    }, [getPreference, keys]);


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

    const handleDeleteMember = () => {
        setDeletingMember(selectedMember);
    }

    const handleExport = () => {
        setExportOpened(true);
    }

    const handleCloseMenu = () => {
        setAnchorEl(null);
    };

    const handleColumnToggle = (key: string) => (event: React.ChangeEvent<HTMLInputElement>) => {
        const visible = event.target.checked;
        const newState = {...columnVisibility, [key]: visible};
        setColumnVisibility(newState);
        gridRef.current?.api.setColumnsVisible([key], visible);
        setPreference("memberTable.columns", newState);
    };

    const handleSaveEdit = async (updated: Member) => {
        const updatedMember = await updateMember(updated);
        onMemberUpdated(updatedMember);
        setEditingMember(null);
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
                    <Tooltip title={t("tooltips.edit")}>
                        <IconButton onClick={handleEditMember} disabled={!selectedMember} color={"primary"}>
                            <EditIcon/>
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t("tooltips.delete")}>
                        <IconButton onClick={handleDeleteMember} disabled={!selectedMember} color={"primary"}>
                            <Delete/>
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t("tooltips.export")}>
                        <IconButton onClick={handleExport} color={"primary"}>
                            <FileDownload/>
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t("tooltips.columnselection")}>
                        <IconButton onClick={handleOpenMenu} color={"primary"}>
                            <ViewColumn/>
                        </IconButton>
                    </Tooltip>
                </Box>

                <Menu anchorEl={anchorEl} open={menuOpen} onClose={handleCloseMenu} sx={{height: "50%"}}>
                    {keys.map((key) => (
                        <MenuItem key={key} dense disableGutters>
                            <FormControlLabel sx={{paddingLeft: 1.5}}
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
            <Box sx={{height: "calc(100vh - 300px)", width: "100%", overflowX: "auto"}}>
                <div style={{minWidth: "1000px", height: "100%"}}>
                    <AgGridReact
                        ref={gridRef}
                        className={themeContext.mode == 'dark' ? 'ag-theme-alpine-dark' : 'ag-theme-alpine'}
                        suppressMovableColumns={true}
                        rowData={members}
                        columnDefs={columnDefs.map((col) => ({
                            ...col,
                            hide: !columnVisibility[col.field ?? ""],
                        }))}
                        theme={themeMaterial}
                        defaultColDef={{sortable: true, filter: true, resizable: true, minWidth: 150}}
                        pagination
                        paginationPageSize={20}
                        rowSelection="single"
                        onSelectionChanged={handleSelectionChanged}
                    />
                </div>
            </Box>
            {editingMember && (
                <EditMemberDialog
                    member={editingMember!}
                    onClose={() => setEditingMember(null)}
                    onSave={(update: Member) => {
                        handleSaveEdit(update);
                    }}
                />
            )}
            {deletingMember && (
                <DeletingMemberDialog
                    member={deletingMember!}
                    onClose={() => setDeletingMember(null)}
                    onDelete={(deleted: Member) => {
                        deleteMember(deleted!).then(() => {
                            onMemberDeleted(deleted!);
                            setDeletingMember(null)
                        })
                    }}/>
            )}
            {exportOpened && (
                <ExportMembersDialog members={members} onClose={() => setExportOpened(false)}/>
            )}
        </Box>
    );
}

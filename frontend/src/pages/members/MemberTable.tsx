import {Backdrop, Box, CircularProgress, IconButton, Tooltip, Typography} from "@mui/material";
import {useTranslation} from "react-i18next";
import React, {useEffect, useMemo, useState} from "react";
import {Member} from "../../api/types";
import {Delete, FileDownload, FileUpload} from "@mui/icons-material";
import {EditMemberDialog} from "./EditMemberDialog";
import EditIcon from '@mui/icons-material/Edit';
import {createMember, deleteMembers, updateMember} from "../../api/members";
import {useUserPreference} from "../../hooks/useUserPreference";
import {DateRenderer, DefaultRenderer, MemberContainingNamedArtifactRenderer} from "./renderer";
import {DeletingMemberDialog} from "./DeletingMemberDialog";
import {ExportMembersDialog} from "./ExportMembersDialog";
import ImportMembersWizard from "./ImportMembersWizard";
import {useTheme} from '@mui/material/styles';
import {DataGrid, GridColDef, GridColumnVisibilityModel, GridRowSelectionModel} from '@mui/x-data-grid';
import type {GridRenderCellParams} from "@mui/x-data-grid/models/params/gridCellParams";


type MemberTableProps = {
    members: Member[];
    onMemberUpdated: (newMember: Member) => void;
    onMembersDeleted: (deletedMember: Member[]) => void;
};

export default function MemberTable({members, onMemberUpdated, onMembersDeleted}: MemberTableProps) {
    const {t} = useTranslation();
    const [editingMember, setEditingMember] = useState<Member | null>(null);
    const [exportOpened, setExportOpened] = useState<boolean>(false);
    const [deletingMembers, setDeletingMembers] = useState<Member[] | null>(null);
    const [selectedMembers, setSelectedMembers] = useState<Member[] | null>(null);
    const {getPreference, setPreference} = useUserPreference();
    const [importWizardOpen, setImportWizardOpen] = useState(false);
    const [importInProgress, setImportInProgress] = useState(false);
    const [importProgress, setImportProgress] = useState(0);
    const [columnVisibility, setColumnVisibility] = useState<GridColumnVisibilityModel>({});

    const keys = useMemo(() => {
        if (!members || members.length === 0) return [];
        return Object.keys(members[0]) as (keyof Member)[];
    }, [members]);
    const theme = useTheme();

    const columnDefs: GridColDef[] = useMemo(() => {
        if (!members || members.length === 0) return [];

        const keys = Object.keys(members[0]) as (keyof Member)[];

        const getCellRenderer = (key: keyof Member) => {
            switch (key) {
                case "entryDate":
                case "exitDate":
                case "birthday":
                    return DateRenderer;
                case "roles":
                case "groups":
                case "sections":
                    return MemberContainingNamedArtifactRenderer;
                default:
                    return DefaultRenderer;
            }
        };

        return keys.map((key) => ({
            field: key,
            headerName: t("members.table.header." + key),
            flex: 1,
            minWidth: 150,
            sortable: true,
            renderCell: (params: GridRenderCellParams) => {
                const Renderer = getCellRenderer(key);
                return <Renderer value={params.value} row={params.row} />;
            }
        }));
    }, [members, t]);


    useEffect(() => {
        if (!keys.length) return;
        getPreference("memberTableColumns").then((stored) => {
            const fallback = Object.fromEntries(keys.map((k, i) => [k, i < 10]));
            const merged = { ...fallback, ...(stored || {}) };
            setColumnVisibility(merged);
        });
    }, [getPreference, keys]);

    const handleSelectionChanged = (selectionModel: GridRowSelectionModel) => {
        const selected = members.filter(member =>
            selectionModel.ids.has(member.id)
        );

        setSelectedMembers(selected.length ? selected : null);
    };

    const handleColumnToggle = (visibilityModel: GridColumnVisibilityModel) => {
        setColumnVisibility(visibilityModel);
        setPreference("memberTableColumns", visibilityModel);
    };

    const handleEditMember = () => {
        if (selectedMembers?.length === 1) {
            setEditingMember(selectedMembers[0])
        }
    }

    const handleDeleteMembers = () => {
        setDeletingMembers(selectedMembers);
    }

    const handleExport = () => {
        setExportOpened(true);
    }

    const handleSaveEdit = async (updated: Member) => {
        const updatedMember = await updateMember(updated);
        onMemberUpdated(updatedMember);
        setEditingMember(null);
    };

    async function upsertImportedMembers(imported: Member[]) {
        const existingMap = new Map(members.map(m => [m.email.toLowerCase(), m]));
        setImportInProgress(true);
        let i = 0;
        const max = imported.length;

        for (const member of imported) {
            const emailKey = member.email.toLowerCase();
            const progress = Math.round((i / max) * 100);
            setImportProgress(progress);
            i++;

            if (existingMap.has(emailKey)) {
                const existing = existingMap.get(emailKey)!;
                const updated = {...existing, ...member, id: existing.id};
                const saved = await updateMember(updated);
                onMemberUpdated(saved);
            } else {
                const saved = await createMember(member);
                onMemberUpdated(saved);
            }
        }
        setImportProgress(100);
        setImportInProgress(false);
    }

    const agGridThemeClass =
        theme.palette.mode === 'dark'
            ? 'ag-theme-clubmanager-dark'
            : 'ag-theme-clubmanager-light';

    return (
        <Box sx={{height: "calc(100vh - 300px)", width: "100%"}}>


            <Box display="flex" justifyContent="space-between" alignItems="center" mb={1}>
                <Typography variant="body1">{t("members.table.description")}</Typography>

                <Box display="flex" justifyContent="end">
                    <Tooltip title={t("tooltips.edit")}>
                        <IconButton onClick={handleEditMember} disabled={selectedMembers?.length !== 1} color={"primary"}>
                            <EditIcon/>
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t("tooltips.delete")}>
                        <IconButton onClick={handleDeleteMembers} disabled={!selectedMembers?.length} color="primary">
                            <Delete/>
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t("tooltips.export")}>
                        <IconButton onClick={handleExport} color={"primary"}>
                            <FileDownload/>
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t("tooltips.import")}>
                        <IconButton onClick={() => setImportWizardOpen(true)} color={"primary"}>
                            <FileUpload/>
                        </IconButton>
                    </Tooltip>

                    {importWizardOpen && (
                        <ImportMembersWizard
                            onClose={() => setImportWizardOpen(false)}
                            onImport={async (imported: Member[]) => {
                                setImportWizardOpen(false);
                                await upsertImportedMembers(imported);
                            }}
                        />
                    )}
                </Box>
            </Box>
            <Box sx={{height: "calc(100vh - 300px)", width: "100%", overflowX: "auto"}}>
                <div style={{minWidth: "1000px", height: "100%"}} className={agGridThemeClass}>

                    <DataGrid
                        autoHeight
                        rows={members}
                        columns={columnDefs.map((col) => ({
                            ...col,
                            sortable: true,
                            filterable: true,
                            minWidth: 150,
                        }))}
                        columnVisibilityModel={columnVisibility}
                        pageSizeOptions={[5, 10, 25]}
                        onRowSelectionModelChange={(selectionModel: GridRowSelectionModel) => {
                            handleSelectionChanged(selectionModel);
                        }}
                        onColumnVisibilityModelChange={(visibilityModel: GridColumnVisibilityModel) => {
                            handleColumnToggle(visibilityModel)
                        }}
                        checkboxSelection />

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
            {deletingMembers && (
                <DeletingMemberDialog
                    members={deletingMembers!}
                    onClose={() => setDeletingMembers(null)}
                    onDelete={(deleted: Member[]) => {
                        deleteMembers(deleted!).then(() => {
                            onMembersDeleted(deleted!);
                            setDeletingMembers(null)
                        })
                    }}/>
            )}
            {exportOpened && (
                <ExportMembersDialog members={members} onClose={() => setExportOpened(false)}/>
            )}
            <Backdrop
                sx={{
                    color: '#fff',
                    zIndex: (theme) => theme.zIndex.drawer + 1
                }}
                open={importInProgress}
            >
                <Box textAlign="center">
                    <CircularProgress color="inherit"/>
                    <Typography sx={{mt: 2}}>{t("members.table.importInProgress")} ({importProgress}%)</Typography>
                </Box>
            </Backdrop>
        </Box>
    );
}

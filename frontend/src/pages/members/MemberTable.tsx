import {
    Backdrop,
    Box,
    Checkbox, CircularProgress,
    FormControlLabel,
    IconButton,
    Menu,
    MenuItem,
    Tooltip,
    Typography
} from "@mui/material";
import {useTranslation} from "react-i18next";
import {AgGridReact} from "ag-grid-react";
import React, {useEffect, useMemo, useRef, useState} from "react";
import {Member} from "../../api/types";
import {AllCommunityModule, ModuleRegistry} from "ag-grid-community";
import {Delete, FileDownload, FileUpload, ViewColumn} from "@mui/icons-material";
import {EditMemberDialog} from "./EditMemberDialog";
import EditIcon from '@mui/icons-material/Edit';
import {createMember, deleteMembers, updateMember} from "../../api/members";
import {useUserPreference} from "../../hooks/useUserPreference";
import {DateRenderer, DefaultRenderer, MemberContainingNamedArtifactRenderer} from "./renderer";
import {DeletingMemberDialog} from "./DeletingMemberDialog";
import {ExportMembersDialog} from "./ExportMembersDialog";
import ImportMembersWizard from "./ImportMembersWizard";
import { useTheme } from '@mui/material/styles';

ModuleRegistry.registerModules([AllCommunityModule]);

type MemberTableProps = {
    members: Member[];
    onMemberUpdated: (newMember: Member) => void;
    onMembersDeleted: (deletedMember: Member[]) => void;
};

export default function MemberTable({members, onMemberUpdated, onMembersDeleted}: MemberTableProps) {
    const {t} = useTranslation();
    const gridRef = useRef<AgGridReact>(null);
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const [editingMember, setEditingMember] = useState<Member | null>(null);
    const [exportOpened, setExportOpened] = useState<boolean>(false);
    const [deletingMembers, setDeletingMembers] = useState<Member[] | null>(null);
    const [selectedMembers, setSelectedMembers] = useState<Member[] | null>(null);
    const {getPreference, setPreference} = useUserPreference();
    const [importWizardOpen, setImportWizardOpen] = useState(false);
    const [importInProgress, setImportInProgress] = useState(false);
    const [importProgress, setImportProgress] = useState(0);

    const keys = useMemo(() => {
        if (!members || members.length === 0) return [];
        return Object.keys(members[0]) as (keyof Member)[];
    }, [members]);
    const theme = useTheme();

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
        const selectedNodes = gridRef.current?.api.getSelectedNodes().map((node) => node.data) || null;
        setSelectedMembers(selectedNodes);
    };

    const handleOpenMenu = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
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

    async function upsertImportedMembers(imported: Member[]) {
        const existingMap = new Map(members.map(m => [m.email.toLowerCase(), m]));

        setImportInProgress(true);

        let i = 0;
        const max = imported.length;
        for (const member of imported) {
            const progress = Math.round((i++ / max) * 100);
            setImportProgress(progress);
            const emailKey = member.email.toLowerCase();
            if (existingMap.has(emailKey)) {
                // Update existing member
                const existing = existingMap.get(emailKey)!;
                const updated = {...existing, ...member, id: existing.id};
                const saved = await updateMember(updated);
                onMemberUpdated(saved);
            } else {
                // create new member
                const saved = await createMember(member);
                onMemberUpdated(saved);
            }
        }
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
                        <IconButton onClick={handleDeleteMembers} disabled={!selectedMembers} color={"primary"}>
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
                <div style={{minWidth: "1000px", height: "100%"}} className={agGridThemeClass}>
                    <AgGridReact
                        headerHeight={100}
                        ref={gridRef}
                        suppressMovableColumns={true}
                        rowData={members}
                        columnDefs={columnDefs.map((col) => ({
                            ...col,
                            hide: !columnVisibility[col.field ?? ""],
                        }))}
                        defaultColDef={{sortable: true, filter: true, resizable: true, minWidth: 150}}
                        pagination
                        paginationPageSize={20}
                        rowSelection="multiple"
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

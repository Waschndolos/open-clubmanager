import { useEffect, useMemo, useRef, useState } from 'react';
import {
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    IconButton,
    InputAdornment,
    InputLabel,
    MenuItem,
    Select,
    TextField,
    Tooltip,
    Typography,
} from '@mui/material';
import { Delete, Description, Download, DriveFolderUpload, Edit, Search } from '@mui/icons-material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useTranslation } from 'react-i18next';
import PageHeader from '../../components/common/PageHeader';
import { deleteDocument, downloadDocument, fetchDocuments, updateDocument, uploadDocument } from '../../api/documents';
import { ClubDocument } from '../../api/types';

type UploadDialogProps = {
    onClose: () => void;
    onSubmit: (data: { title: string; category: string; description: string; file: File }) => Promise<void>;
};

function UploadDialog({ onClose, onSubmit }: UploadDialogProps) {
    const { t } = useTranslation();
    const fileInputRef = useRef<HTMLInputElement | null>(null);
    const [title, setTitle] = useState('');
    const [category, setCategory] = useState('');
    const [description, setDescription] = useState('');
    const [file, setFile] = useState<File | null>(null);
    const [dragActive, setDragActive] = useState(false);

    const pickFile = () => fileInputRef.current?.click();
    const applyFile = (pickedFile: File | null) => {
        if (!pickedFile) {
            return;
        }
        setFile(pickedFile);
        if (!title.trim()) {
            setTitle(pickedFile.name);
        }
    };

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{t('documents.upload.title')}</DialogTitle>
            <DialogContent>
                <Box mt={1} display="flex" flexDirection="column" gap={2}>
                    <TextField label={t('documents.fields.title')} value={title} onChange={(e) => setTitle(e.target.value)} fullWidth />
                    <TextField label={t('documents.fields.category')} value={category} onChange={(e) => setCategory(e.target.value)} fullWidth />
                    <TextField
                        label={t('documents.fields.description')}
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        multiline
                        minRows={2}
                        fullWidth
                    />
                    <Box
                        onClick={pickFile}
                        onDragOver={(e) => {
                            e.preventDefault();
                            setDragActive(true);
                        }}
                        onDragLeave={() => setDragActive(false)}
                        onDrop={(e) => {
                            e.preventDefault();
                            setDragActive(false);
                            applyFile(e.dataTransfer.files[0] ?? null);
                        }}
                        sx={{
                            border: '2px dashed',
                            borderColor: dragActive ? 'primary.main' : 'divider',
                            borderRadius: 2,
                            p: 3,
                            textAlign: 'center',
                            cursor: 'pointer',
                            bgcolor: dragActive ? 'action.hover' : 'background.paper',
                        }}
                    >
                        <DriveFolderUpload color="primary" />
                        <Typography variant="body2" mt={1}>
                            {file ? file.name : t('documents.upload.dropzone')}
                        </Typography>
                    </Box>
                    <input
                        ref={fileInputRef}
                        type="file"
                        hidden
                        onChange={(e) => applyFile(e.target.files?.[0] ?? null)}
                    />
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button
                    variant="contained"
                    onClick={() => file && void onSubmit({ title, category, description, file })}
                    disabled={!file || !title.trim() || !category.trim()}
                >
                    {t('documents.upload.submit')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}

export default function Documents() {
    const { t } = useTranslation();
    const [documents, setDocuments] = useState<ClubDocument[]>([]);
    const [error, setError] = useState<string | null>(null);
    const [search, setSearch] = useState('');
    const [categoryFilter, setCategoryFilter] = useState('ALL');
    const [uploadOpen, setUploadOpen] = useState(false);
    const [editingDocument, setEditingDocument] = useState<ClubDocument | null>(null);

    useEffect(() => {
        void fetchDocuments()
            .then((rows) => {
                setDocuments(rows);
                setError(null);
            })
            .catch(() => setError(t('documents.loadError')));
    }, [t]);

    const categories = useMemo(
        () => Array.from(new Set(documents.map((document) => document.category))).sort((a, b) => a.localeCompare(b)),
        [documents]
    );

    const filteredDocuments = useMemo(() => {
        const query = search.trim().toLowerCase();
        return documents.filter((document) => {
            const matchesCategory = categoryFilter === 'ALL' || document.category === categoryFilter;
            const matchesSearch =
                !query ||
                document.title.toLowerCase().includes(query) ||
                document.filename.toLowerCase().includes(query) ||
                document.category.toLowerCase().includes(query) ||
                (document.description ?? '').toLowerCase().includes(query);
            return matchesCategory && matchesSearch;
        });
    }, [categoryFilter, documents, search]);

    const columns: GridColDef[] = [
        { field: 'title', headerName: t('documents.fields.title'), flex: 1.2, minWidth: 160 },
        { field: 'category', headerName: t('documents.fields.category'), flex: 0.8, minWidth: 120 },
        { field: 'filename', headerName: t('documents.fields.filename'), flex: 1.2, minWidth: 180 },
        {
            field: 'size',
            headerName: t('documents.fields.size'),
            width: 130,
            valueFormatter: (value: number) => `${(value / 1024 / 1024).toFixed(2)} MB`,
        },
        {
            field: 'createdAt',
            headerName: t('documents.fields.createdAt'),
            width: 140,
            valueFormatter: (value: string) => new Date(value).toLocaleDateString(),
        },
        {
            field: 'actions',
            headerName: t('members.table.header.actions'),
            sortable: false,
            width: 140,
            renderCell: (params) => {
                const row = params.row as ClubDocument;
                return (
                    <Box>
                        <Tooltip title={t('documents.actions.download')}>
                            <IconButton size="small" onClick={() => void downloadDocument(row.id, row.filename)}>
                                <Download fontSize="small" />
                            </IconButton>
                        </Tooltip>
                        <Tooltip title={t('tooltips.edit')}>
                            <IconButton size="small" onClick={() => setEditingDocument(row)}>
                                <Edit fontSize="small" />
                            </IconButton>
                        </Tooltip>
                        <Tooltip title={t('tooltips.delete')}>
                            <IconButton
                                size="small"
                                onClick={() => {
                                    void deleteDocument(row.id)
                                        .then(() => setDocuments((prev) => prev.filter((item) => item.id !== row.id)))
                                        .catch(() => setError(t('documents.deleteError')));
                                }}
                            >
                                <Delete fontSize="small" />
                            </IconButton>
                        </Tooltip>
                    </Box>
                );
            },
        },
    ];

    return (
        <Box p={3}>
            <PageHeader
                title={t('documents.title')}
                icon={<Description fontSize="small" />}
                actions={
                    <Button variant="contained" startIcon={<DriveFolderUpload />} onClick={() => setUploadOpen(true)}>
                        {t('documents.upload.open')}
                    </Button>
                }
            />
            {error && <Typography color="error" mb={2}>{error}</Typography>}
            <Box display="flex" gap={2} mb={2} flexWrap="wrap">
                <TextField
                    label={t('documents.search')}
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    size="small"
                    slotProps={{
                        input: {
                            startAdornment: (
                                <InputAdornment position="start">
                                    <Search fontSize="small" />
                                </InputAdornment>
                            ),
                        },
                    }}
                />
                <FormControl size="small" sx={{ minWidth: 220 }}>
                    <InputLabel>{t('documents.categoryFilter')}</InputLabel>
                    <Select
                        label={t('documents.categoryFilter')}
                        value={categoryFilter}
                        onChange={(e) => setCategoryFilter(e.target.value)}
                    >
                        <MenuItem value="ALL">{t('documents.categoryAll')}</MenuItem>
                        {categories.map((category) => (
                            <MenuItem key={category} value={category}>{category}</MenuItem>
                        ))}
                    </Select>
                </FormControl>
            </Box>

            <DataGrid autoHeight rows={filteredDocuments} columns={columns} pageSizeOptions={[10, 25, 50]} />

            {uploadOpen && (
                <UploadDialog
                    onClose={() => setUploadOpen(false)}
                    onSubmit={async ({ title, category, description, file }) => {
                        try {
                            const created = await uploadDocument({ title, category, description, file });
                            setDocuments((prev) => [created, ...prev]);
                            setUploadOpen(false);
                            setError(null);
                        } catch {
                            setError(t('documents.uploadError'));
                        }
                    }}
                />
            )}

            {editingDocument && (
                <Dialog open onClose={() => setEditingDocument(null)} fullWidth maxWidth="sm">
                    <DialogTitle>{t('documents.edit.title')}</DialogTitle>
                    <DialogContent>
                        <Box mt={1} display="flex" flexDirection="column" gap={2}>
                            <TextField
                                label={t('documents.fields.title')}
                                value={editingDocument.title}
                                onChange={(e) => setEditingDocument((prev) => prev ? { ...prev, title: e.target.value } : prev)}
                                fullWidth
                            />
                            <TextField
                                label={t('documents.fields.category')}
                                value={editingDocument.category}
                                onChange={(e) => setEditingDocument((prev) => prev ? { ...prev, category: e.target.value } : prev)}
                                fullWidth
                            />
                            <TextField
                                label={t('documents.fields.description')}
                                value={editingDocument.description ?? ''}
                                onChange={(e) => setEditingDocument((prev) => prev ? { ...prev, description: e.target.value } : prev)}
                                multiline
                                minRows={2}
                                fullWidth
                            />
                        </Box>
                    </DialogContent>
                    <DialogActions>
                        <Button onClick={() => setEditingDocument(null)}>{t('buttons.abort')}</Button>
                        <Button
                            variant="contained"
                            onClick={() => {
                                void updateDocument(editingDocument.id, {
                                    title: editingDocument.title,
                                    category: editingDocument.category,
                                    description: editingDocument.description ?? '',
                                })
                                    .then((updated) => {
                                        setDocuments((prev) => prev.map((item) => item.id === updated.id ? updated : item));
                                        setEditingDocument(null);
                                    })
                                    .catch(() => setError(t('documents.updateError')));
                            }}
                            disabled={!editingDocument.title.trim() || !editingDocument.category.trim()}
                        >
                            {t('buttons.save')}
                        </Button>
                    </DialogActions>
                </Dialog>
            )}
        </Box>
    );
}

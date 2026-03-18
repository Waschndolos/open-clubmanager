import { useEffect, useState } from 'react';
import {
    Alert,
    Box,
    Chip,
    CircularProgress,
    Typography,
} from '@mui/material';
import { History as HistoryIcon } from '@mui/icons-material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useTranslation } from 'react-i18next';
import { fetchHistory, AuditLog } from '../../api/history';
import PageHeader from '../../components/common/PageHeader';

function ActionChip({ action }: { action: AuditLog['action'] }) {
    const colorMap: Record<AuditLog['action'], 'success' | 'warning' | 'error'> = {
        CREATE: 'success',
        UPDATE: 'warning',
        DELETE: 'error',
    };
    return <Chip label={action} color={colorMap[action]} size="small" />;
}

export default function History() {
    const { t } = useTranslation();
    const [logs, setLogs] = useState<AuditLog[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        fetchHistory()
            .then(setLogs)
            .catch((err) => {
                console.error(err);
                setError(t('history.loadError'));
            })
            .finally(() => setLoading(false));
    }, [t]);

    const columns: GridColDef[] = [
        {
            field: 'createdAt',
            headerName: t('history.table.timestamp'),
            width: 200,
            valueFormatter: (value: string) =>
                value ? new Date(value).toLocaleString() : '',
        },
        {
            field: 'action',
            headerName: t('history.table.action'),
            width: 110,
            renderCell: (params) => <ActionChip action={params.value as AuditLog['action']} />,
        },
        {
            field: 'entity',
            headerName: t('history.table.entity'),
            width: 180,
        },
        {
            field: 'entityId',
            headerName: t('history.table.entityId'),
            width: 90,
        },
        {
            field: 'userId',
            headerName: t('history.table.user'),
            flex: 1,
            minWidth: 200,
        },
        {
            field: 'data',
            headerName: t('history.table.details'),
            flex: 2,
            minWidth: 200,
            renderCell: (params) => {
                if (!params.value) return null;
                try {
                    const parsed = JSON.parse(params.value as string);
                    return (
                        <Typography variant="caption" noWrap title={JSON.stringify(parsed, null, 2)}>
                            {Object.entries(parsed)
                                .map(([k, v]) => `${k}: ${v}`)
                                .join(', ')}
                        </Typography>
                    );
                } catch {
                    return <Typography variant="caption">{params.value as string}</Typography>;
                }
            },
        },
    ];

    return (
        <>
            <PageHeader
                title={t('menu.history')}
                icon={<HistoryIcon fontSize="small" />}
            />
            {loading ? (
                <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
                    <CircularProgress />
                </Box>
            ) : error ? (
                <Alert severity="error">{error}</Alert>
            ) : (
                <DataGrid
                    rows={logs}
                    columns={columns}
                    pageSizeOptions={[25, 50, 100]}
                    initialState={{ pagination: { paginationModel: { pageSize: 25 } } }}
                    disableRowSelectionOnClick
                    sx={{ border: 'none' }}
                />
            )}
        </>
    );
}

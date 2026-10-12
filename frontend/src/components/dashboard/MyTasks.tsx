import { useEffect, useState } from 'react';
import { Box, ButtonBase, Chip, Paper, Typography } from '@mui/material';
import TaskAltOutlined from '@mui/icons-material/TaskAltOutlined';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { fetchTasks, Task } from '../../api/tasks';

const SHOWN = 5;

/** Dashboard tile "My tasks" with overdue tasks highlighted; hidden when there are none. */
export default function MyTasks() {
    const { t, i18n } = useTranslation();
    const navigate = useNavigate();
    const [list, setList] = useState<Task[]>([]);

    useEffect(() => {
        fetchTasks('mine').then(setList).catch(() => setList([]));
    }, []);

    if (list.length === 0) return null;

    return (
        <Box component="section" sx={{ mt: { xs: 4, sm: 5 } }}>
            <Typography variant="h6" component="h2" sx={{ mb: 1.5 }}>{t('tasks.dashboard.title')}</Typography>
            <Paper variant="outlined" sx={{ borderRadius: 3.5, p: 1 }}>
                {list.slice(0, SHOWN).map((task) => (
                    <Box key={task.id} sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 1.5, py: 1 }}>
                        <TaskAltOutlined color={task.overdue ? 'error' : 'primary'} fontSize="small" />
                        <Typography sx={{ flexGrow: 1, fontWeight: 600 }}>{task.title}</Typography>
                        {task.overdue && <Chip size="small" color="error" label={t('tasks.overdue')} />}
                        {task.dueDate && (
                            <Typography color="text.secondary" sx={{ fontSize: '0.9rem' }}>
                                {new Date(`${task.dueDate}T00:00:00`).toLocaleDateString(i18n.language)}
                            </Typography>
                        )}
                    </Box>
                ))}
                <ButtonBase onClick={() => navigate('/tasks')} sx={{ px: 1.5, py: 1, color: 'primary.main', fontWeight: 600 }}>
                    {t('tasks.dashboard.all', { count: list.length })}
                </ButtonBase>
            </Paper>
        </Box>
    );
}

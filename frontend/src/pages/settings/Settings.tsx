import {
  Container,
  Typography,
  TextField,
  Select,
  MenuItem,
  Button,
  FormControl,
  InputLabel,
  Paper,
  Fade,
  Box,
  CircularProgress,
  Snackbar
} from '@mui/material';
import { useTranslation } from 'react-i18next';

export const Settings = () => {
  const { t, i18n } = useTranslation();

  const [dbPath, setDbPath] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(false);
  const [language, setLanguage] = useState<string>('en');
  const [snackBarState, setsnackBarState] = useState<{open: boolean, message:string}>({
    open: false,
    message: ""
  });

  const availableLanguages = () => {
    return Object.keys(i18n.services.resourceStore.data);
  }

  const fetchPreference = () => {

  }

  useEffect(() => {
    const savedLanguage = localStorage.getItem('language');

    if (savedLanguage) {
      setLanguage(savedLanguage);
      i18n.changeLanguage(savedLanguage);
    }
  }, [i18n]);

  const saveSettings = () => {
    localStorage.setItem('language', language);
    i18n.changeLanguage(language);

    setsnackBarState({
      open: true,
      message: t("settings.labels.saveSuccess")
    });
  };

  if (loading) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        height="100vh"
      >
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 4 }}>
      <Typography variant="h4" gutterBottom>
          {t("settings.title")}
      </Typography>

      <Paper sx={{ p: 3 }}>

        <FormControl fullWidth margin="normal">
          <TextField 
            label={t("settings.labels.dbPath")}
            variant="outlined"
            value={dbPath}
            fullWidth
            onChange={(e) => setDbPath(e.target.value)}
            placeholder="/path/to/database.db"
            margin="normal"
          />
        </FormControl>

        <FormControl fullWidth margin="normal">
          <InputLabel id="language-select-label">{t("settings.labels.dbPath")}</InputLabel>
          <Select
            labelId="language-select-label"
            value={language}
            label={t("settings.labels.dbPath")}
            onChange={(e) => {
              const lng = e.target.value;
              setLanguage(lng);
            }}
          >
            {availableLanguages().map((lng) => (
              <MenuItem key={lng} value={lng}>
                {t(`settings.languages.${lng}`) || lng}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Paper>

      <Button
        variant="contained"
        color="primary"
        onClick={saveSettings}
        sx={{ mt: 2 }}
      >
        Save Settings
      </Button>

      <Snackbar
        open={snackBarState.open}
        color="primary"
        onClose={() => setsnackBarState({ ...snackBarState, open: false })}
        TransitionComponent={Fade}
        message={snackBarState.message}
        autoHideDuration={3000}
      />

    </Box>
  );
};

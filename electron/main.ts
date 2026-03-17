import { app, BrowserWindow, dialog, ipcMain } from 'electron';
import Store from 'electron-store';
import { fileURLToPath } from 'url';
import * as path from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const isDev = !app.isPackaged;
const iconPath = isDev
    ? path.join(__dirname, '..', '..', 'assets', 'clubmanager-icon.ico')
    : path.join(process.resourcesPath, 'assets', 'clubmanager-icon.ico');

const preloadPath = path.join(__dirname, 'preload.cjs');

let mainWindow: BrowserWindow | null = null;

const store = new Store();
const DB_PATH_STORE_KEY = 'database.path';

ipcMain.on('userpreference-get', async (event, userId, key) => {
  event.returnValue = store.get(`${key}-${userId}`);
});

ipcMain.on('userpreference-set', async (event, userId, key, value) => {
  store.set(`${key}-${userId}`, value);
});

ipcMain.on('userpreference-delete', async (event, userId, key, value) => {
  store.set(`${key}-${userId}`, value);
});

ipcMain.on('apppreference-get', async (event, key) => {
  const res = await fetch(`http://localhost:3001/api/preference/app/${key}`);
  const data = await res.json();
  
  event.returnValue = data[key];
});

ipcMain.on('apppreference-set', async (event, key, value) => {

  const response = await fetch(`http://localhost:3001/api/preference/app/${key}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ value })
  });

});

// IPC handlers for database path management
ipcMain.on('db-path-get', (event) => {
  event.returnValue = store.get(DB_PATH_STORE_KEY) ?? null;
});

ipcMain.on('db-path-set', (_event, dbPath: string) => {
  store.set(DB_PATH_STORE_KEY, dbPath);
});

ipcMain.handle('dialog:select-db-directory', async () => {
  const result = await dialog.showOpenDialog(mainWindow!, {
    title: 'Select folder for database',
    properties: ['openDirectory', 'createDirectory'],
  });
  if (result.canceled || result.filePaths.length === 0) return null;
  return result.filePaths[0];
});

ipcMain.handle('dialog:select-db-file', async () => {
  const result = await dialog.showOpenDialog(mainWindow!, {
    title: 'Select existing database file',
    filters: [{ name: 'SQLite Database', extensions: ['db', 'sqlite', 'sqlite3'] }],
    properties: ['openFile'],
  });
  if (result.canceled || result.filePaths.length === 0) return null;
  return result.filePaths[0];
});

async function configureBackendDb(dbPath: string): Promise<void> {
  try {
    await fetch('http://localhost:3001/api/settings/set-db-path', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ dbPath }),
    });
  } catch (err) {
    console.error('Failed to configure backend database path:', err);
  }
}

async function waitForBackend(retries = 20, delayMs = 500): Promise<void> {
  for (let i = 0; i < retries; i++) {
    try {
      const res = await fetch('http://localhost:3001/api/settings/db-status');
      if (res.ok) return;
    } catch {
      // not ready yet
    }
    await new Promise((resolve) => setTimeout(resolve, delayMs));
  }
  console.warn('Backend did not become ready in time.');
}

const gotTheLock = app.requestSingleInstanceLock();

if (!gotTheLock) {
    app.quit();
} else {
    app.on('second-instance', () => {
        // Wenn die zweite Instanz gestartet wird, bringe das Hauptfenster in den Vordergrund
        if (mainWindow) {
            if (mainWindow.isMinimized()) mainWindow.restore();
            mainWindow.focus();
        }
    });

    app.whenReady().then(async () => {
        // Wait for the backend to start, then restore stored database path if available.
        await waitForBackend();
        const storedDbPath = store.get(DB_PATH_STORE_KEY) as string | undefined;
        if (storedDbPath) {
            await configureBackendDb(storedDbPath);
        }
        createWindow();
    });

    app.on('window-all-closed', () => {
        if (process.platform !== 'darwin') {
            app.quit();
        }
    });

    app.on('activate', () => {
        if (BrowserWindow.getAllWindows().length === 0) {
            createWindow();
        }
    });
}

function createWindow() {
    if (mainWindow !== null) return;

    mainWindow = new BrowserWindow({
        width: 800,
        height: 600,
        icon: iconPath,
        webPreferences: {
            contextIsolation: true,
            preload: preloadPath
        },
    });


    if (isDev) {
        console.log(`Starting in dev mode`)
    }

    if (isDev) {
        mainWindow.loadURL('http://localhost:5173');
    } else {
        const indexPath = path.join(__dirname, '..', '..', '..', 'frontend', 'dist', 'index.html');
        mainWindow.loadFile(indexPath);
    }


    mainWindow.on('closed', () => {
        mainWindow = null;
    });
}

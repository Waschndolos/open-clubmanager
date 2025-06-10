import { app, BrowserWindow, ipcMain } from 'electron';
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

ipcMain.on('electron-store-get', async (event, val) => {
  event.returnValue = store.get(val);
});
ipcMain.on('electron-store-set', async (event, key, val) => {
  store.set(key, val);
});

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

    app.whenReady().then(() => {
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

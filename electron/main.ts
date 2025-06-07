import {app, BrowserWindow} from 'electron';
import * as path from 'path';

const isDev = !app.isPackaged;
const iconPath = isDev
    ? path.join(__dirname, '..', '..', 'assets', 'clubmanager-icon.ico')
    : path.join(process.resourcesPath, 'assets', 'clubmanager-icon.ico');

let mainWindow: BrowserWindow | null = null;

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
        },
    });


    if (isDev) {
        console.log(`Starting in dev mode`)
    }

    const indexPath = isDev
        ? path.join(__dirname, 'frontend', 'dist', 'index.html')
        : path.join(__dirname, '..', '..', 'frontend', 'dist', 'index.html');


    mainWindow.loadFile(indexPath).then(() =>console.log("Successfully loaded index.html")).catch(err => {
        console.error('❌ Failed to load index.html:', err);
    });


    mainWindow.on('closed', () => {
        mainWindow = null;
    });
}

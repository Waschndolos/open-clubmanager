import { app, BrowserWindow, ipcMain, dialog } from 'electron';
import Store from 'electron-store';
import { fileURLToPath } from 'url';
import * as path from 'path';
import * as os from 'os';
import { FolderEventStore } from './storage/FolderEventStore.js';
import { LockInfo } from './storage/types.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const isDev = !app.isPackaged;
const iconPath = isDev
    ? path.join(__dirname, '..', '..', 'assets', 'clubmanager-icon.ico')
    : path.join(process.resourcesPath, 'assets', 'clubmanager-icon.ico');

const preloadPath = path.join(__dirname, 'preload.cjs');

let mainWindow: BrowserWindow | null = null;

const store = new Store();

let folderStore: FolderEventStore | null = null;

function getActor(): string {
    return os.userInfo().username;
}

function getFolderStore(): FolderEventStore {
    if (!folderStore) {
        throw new Error('Club folder not configured. Please select a club folder first.');
    }
    return folderStore;
}

function initFolderStore(folderPath: string): void {
    folderStore = new FolderEventStore(folderPath);
    folderStore.ensureFolderStructure();
}

// ── User preferences (existing) ──────────────────────────────────────────────

ipcMain.on('userpreference-get', async (event, userId, key) => {
    event.returnValue = store.get(`${key}-${userId}`);
});

ipcMain.on('userpreference-set', async (event, userId, key, value) => {
    store.set(`${key}-${userId}`, value);
});

ipcMain.on('userpreference-delete', async (event, userId, key, value) => {
    store.set(`${key}-${userId}`, value);
});

ipcMain.on('apppreference-get', (event, key) => {
    // Use electron-store as local storage for app preferences in Folder Mode.
    // This avoids crashing when the HTTP backend is not running.
    event.returnValue = store.get(`apppreference-${key}`) ?? null;
});

ipcMain.on('apppreference-set', (_event, key, value) => {
    store.set(`apppreference-${key}`, value);
    // Best-effort sync to backend when available (non-blocking).
    fetch(`http://localhost:3001/api/preference/app/${key}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ value }),
    }).catch(() => {
        // Backend not available – preference is already stored locally.
    });
});

// ── Club IPC handlers ─────────────────────────────────────────────────────────

ipcMain.handle('club:selectFolder', async () => {
    const win = mainWindow ?? BrowserWindow.getFocusedWindow();
    if (!win) return null;
    const result = await dialog.showOpenDialog(win, {
        properties: ['openDirectory'],
        title: 'Select Club Data Folder',
    });
    if (result.canceled || result.filePaths.length === 0) return null;
    const folderPath = result.filePaths[0];
    store.set('clubFolder', folderPath);
    initFolderStore(folderPath);
    return folderPath;
});

ipcMain.handle('club:getFolder', () => {
    return store.get('clubFolder') ?? null;
});

ipcMain.handle('club:initFolder', () => {
    const folderPath = store.get('clubFolder') as string | undefined;
    if (!folderPath) throw new Error('No club folder configured.');
    initFolderStore(folderPath);
    return { ok: true };
});

// ── Members IPC handlers ──────────────────────────────────────────────────────

ipcMain.handle('members:list', () => {
    const state = getFolderStore().getState();
    const members = state['member'] ?? {};
    return Object.values(members);
});

ipcMain.handle('members:get', (_event, id: number) => {
    const state = getFolderStore().getState();
    const member = (state['member'] ?? {})[id];
    if (!member) throw new Error(`Member ${id} not found.`);
    return member;
});

ipcMain.handle('members:create', (_event, data: Record<string, unknown>) => {
    const store_ = getFolderStore();
    const id = store_.generateId();
    const patch = { ...data, id };
    store_.appendEvent('member.create', 'member', id, patch, getActor());
    return patch;
});

ipcMain.handle('members:update', (_event, { id, patch }: { id: number; patch: Record<string, unknown> }) => {
    const store_ = getFolderStore();
    const state = store_.getState();
    const current = (state['member'] ?? {})[id];
    if (!current) throw new Error(`Member ${id} not found.`);
    store_.appendEvent('member.update', 'member', id, patch, getActor());
    return { ...current, ...patch };
});

ipcMain.handle('members:delete', (_event, ids: number[]) => {
    const store_ = getFolderStore();
    for (const id of ids) {
        store_.appendEvent('member.delete', 'member', id, {}, getActor());
    }
    return { ok: true };
});

ipcMain.handle('members:lock', (_event, { id, owner }: { id: number; owner: string }) => {
    const acquired = getFolderStore().locks.acquire(`member-${id}`, owner);
    if (!acquired) {
        const lock = getFolderStore().locks.getLock(`member-${id}`);
        return { acquired: false, lock };
    }
    return { acquired: true, lock: null };
});

ipcMain.handle('members:unlock', (_event, { id, owner }: { id: number; owner: string }) => {
    const released = getFolderStore().locks.release(`member-${id}`, owner);
    return { released };
});

ipcMain.handle('members:getLock', (_event, id: number) => {
    const lock: LockInfo | null = getFolderStore().locks.getLock(`member-${id}`);
    return lock;
});

// ── Entity IPC handlers (roles, groups, sections) ────────────────────────────

function entityHandlers(kind: string) {
    ipcMain.handle(`${kind}:list`, () => {
        const state = getFolderStore().getState();
        const entities = state[kind] ?? {};
        return Object.values(entities);
    });

    ipcMain.handle(`${kind}:create`, (_event, data: Record<string, unknown>) => {
        const store_ = getFolderStore();
        const id = store_.generateId();
        const patch = { ...data, id };
        store_.appendEvent(`${kind}.create`, kind, id, patch, getActor());
        return patch;
    });

    ipcMain.handle(`${kind}:update`, (_event, data: Record<string, unknown>) => {
        const store_ = getFolderStore();
        const id = data.id as number;
        store_.appendEvent(`${kind}.update`, kind, id, data, getActor());
        return data;
    });

    ipcMain.handle(`${kind}:delete`, (_event, id: number) => {
        getFolderStore().appendEvent(`${kind}.delete`, kind, id, {}, getActor());
        return { ok: true };
    });
}

entityHandlers('role');
entityHandlers('group');
entityHandlers('section');

// ── Finance IPC handlers ──────────────────────────────────────────────────────

const KIND_TRANSACTION = 'transaction';
const KIND_MEMBER_FEE = 'memberfee';

ipcMain.handle('finance:listTransactions', () => {
    const state = getFolderStore().getState();
    const txs = state[KIND_TRANSACTION] ?? {};
    return Object.values(txs);
});

ipcMain.handle('finance:createTransaction', (_event, data: Record<string, unknown>) => {
    const store_ = getFolderStore();
    const id = store_.generateId();
    const patch = { ...data, id, createdAt: new Date().toISOString() };
    store_.appendEvent(`${KIND_TRANSACTION}.create`, KIND_TRANSACTION, id, patch, getActor());
    return patch;
});

ipcMain.handle('finance:updateTransaction', (_event, data: Record<string, unknown>) => {
    const store_ = getFolderStore();
    const id = data.id as number;
    const patch = { ...data, updatedAt: new Date().toISOString() };
    store_.appendEvent(`${KIND_TRANSACTION}.update`, KIND_TRANSACTION, id, patch, getActor());
    return patch;
});

ipcMain.handle('finance:deleteTransaction', (_event, id: number) => {
    getFolderStore().appendEvent(`${KIND_TRANSACTION}.delete`, KIND_TRANSACTION, id, {}, getActor());
    return { ok: true };
});

ipcMain.handle('finance:listMemberFees', () => {
    const state = getFolderStore().getState();
    const fees = state[KIND_MEMBER_FEE] ?? {};
    return Object.values(fees);
});

ipcMain.handle('finance:createMemberFee', (_event, data: Record<string, unknown>) => {
    const store_ = getFolderStore();
    const id = store_.generateId();
    const patch = { ...data, id, createdAt: new Date().toISOString() };
    store_.appendEvent(`${KIND_MEMBER_FEE}.create`, KIND_MEMBER_FEE, id, patch, getActor());
    return patch;
});

ipcMain.handle('finance:updateMemberFee', (_event, data: Record<string, unknown>) => {
    const store_ = getFolderStore();
    const id = data.id as number;
    const patch = { ...data, updatedAt: new Date().toISOString() };
    store_.appendEvent(`${KIND_MEMBER_FEE}.update`, KIND_MEMBER_FEE, id, patch, getActor());
    return patch;
});

ipcMain.handle('finance:deleteMemberFee', (_event, id: number) => {
    getFolderStore().appendEvent(`${KIND_MEMBER_FEE}.delete`, KIND_MEMBER_FEE, id, {}, getActor());
    return { ok: true };
});

// ── History IPC handler ───────────────────────────────────────────────────────

ipcMain.handle('history:list', () => {
    const events = getFolderStore().readAllEvents();
    return events.map((ev, index) => ({
        id: index + 1,
        action: ev.type.endsWith('.create')
            ? 'CREATE'
            : ev.type.endsWith('.delete')
                ? 'DELETE'
                : 'UPDATE',
        entity: ev.entity.kind,
        entityId: ev.entity.id,
        userId: ev.actor,
        data: Object.keys(ev.patch).length > 0 ? JSON.stringify(ev.patch) : null,
        createdAt: ev.ts,
    }));
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
        // Restore folder store from persisted path on startup
        const savedFolder = store.get('clubFolder') as string | undefined;
        if (savedFolder) {
            try {
                initFolderStore(savedFolder);
            } catch {
                // folder may no longer exist; user will be prompted again
            }
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

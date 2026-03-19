import { app, BrowserWindow, ipcMain, dialog, shell } from 'electron';
import Store from 'electron-store';
import { fileURLToPath } from 'url';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import * as crypto from 'crypto';
import { FolderEventStore } from './storage/FolderEventStore.js';
import { LockInfo, AttachmentRecord, PaymentRecord } from './storage/types.js';
import { GlobalLockService } from './storage/GlobalLockService.js';
import { SqliteDbService } from './storage/SqliteDbService.js';
import { AttachmentService } from './storage/AttachmentService.js';
import { createBackup } from './storage/BackupService.js';

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
let lockService: GlobalLockService | null = null;
let dbService: SqliteDbService | null = null;
let attachmentService: AttachmentService | null = null;
let lockHeartbeatTimer: ReturnType<typeof setInterval> | null = null;
let currentDataDir: string | null = null;

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

    currentDataDir = folderPath;
    lockService = new GlobalLockService(folderPath, app.getVersion());

    // SQLite and attachment services are optional: if better-sqlite3 is not
    // installed or not compiled for this Electron version, the app continues
    // to work with the event-based storage.  Only payments/attachments/backup
    // features will be unavailable until the module issue is resolved.
    try {
        dbService = new SqliteDbService(folderPath);
        attachmentService = new AttachmentService(folderPath);
        attachmentService.ensureDirs();
        // Open in read-only mode initially; the user must invoke
        // storage:requestEditMode to acquire the write lock and switch to RW.
        dbService.open(true);
    } catch (err) {
        console.warn(
            'SQLite storage backend could not be initialized:',
            err,
            '\nTo fix this, run "npm run electron:rebuild" from the project root and restart the app.',
            '\nMember/role/group/section data saved via the event store remains fully functional.'
        );
        dbService = null;
        attachmentService = null;
    }
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

ipcMain.on('apppreference-get', async (event, key) => {
    const res = await fetch(`http://localhost:3001/api/preference/app/${key}`);
    const data = await res.json();

    event.returnValue = data[key];
});

ipcMain.on('apppreference-set', async (event, key, value) => {

    await fetch(`http://localhost:3001/api/preference/app/${key}`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ value })
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

// ── SQLite storage IPC handlers ───────────────────────────────────────────────

function getDbService(): SqliteDbService {
    if (!dbService) {
        throw new Error('Club folder not configured. Please select a club folder first.');
    }
    return dbService;
}

function getLockService(): GlobalLockService {
    if (!lockService) {
        throw new Error('Club folder not configured. Please select a club folder first.');
    }
    return lockService;
}

function getAttachmentService(): AttachmentService {
    if (!attachmentService) {
        throw new Error('Club folder not configured. Please select a club folder first.');
    }
    return attachmentService;
}

function startLockHeartbeat(): void {
    stopLockHeartbeat();
    lockHeartbeatTimer = setInterval(() => {
        try {
            lockService?.refresh();
        } catch (err) {
            // Non-fatal: heartbeat failure means the lock may go stale,
            // but we don't want to crash the app over it.
            console.warn('Lock heartbeat refresh failed:', err);
        }
    }, 30_000);
}

function stopLockHeartbeat(): void {
    if (lockHeartbeatTimer !== null) {
        clearInterval(lockHeartbeatTimer);
        lockHeartbeatTimer = null;
    }
}

ipcMain.handle('storage:getStatus', () => {
    if (!lockService || !currentDataDir) {
        return { dataDir: null, mode: 'readonly' };
    }
    return lockService.getStatus(currentDataDir);
});

ipcMain.handle('storage:requestEditMode', () => {
    const ls = getLockService();
    const db = getDbService();
    const acquired = ls.tryAcquire();
    if (acquired) {
        db.open(false);
        startLockHeartbeat();
    }
    return { acquired, status: ls.getStatus(currentDataDir!) };
});

ipcMain.handle('storage:releaseEditMode', () => {
    stopLockHeartbeat();
    const ls = getLockService();
    const db = getDbService();
    ls.release();
    db.open(true);
    return { ok: true };
});

ipcMain.handle('storage:exportBackup', async () => {
    if (!currentDataDir) {
        throw new Error('Club folder not configured.');
    }
    const zipPath = await createBackup(currentDataDir);
    return { zipPath };
});

// ── Payments IPC handlers ─────────────────────────────────────────────────────

function dbPaymentToRecord(p: import('./storage/SqliteDbService.js').DbPayment): PaymentRecord {
    return {
        id: p.id,
        memberId: p.member_id,
        amountCents: p.amount_cents,
        currency: p.currency,
        date: p.date,
        note: p.note,
        createdAt: p.created_at,
    };
}

ipcMain.handle('payments:list', () => {
    return getDbService().listPayments().map(dbPaymentToRecord);
});

ipcMain.handle('payments:create', (_event, data: Omit<PaymentRecord, 'id' | 'createdAt'>) => {
    const row = getDbService().createPayment({
        id: crypto.randomUUID(),
        member_id: data.memberId,
        amount_cents: data.amountCents,
        currency: data.currency,
        date: data.date,
        note: data.note,
    });
    return dbPaymentToRecord(row);
});

ipcMain.handle('payments:update', (_event, data: PaymentRecord) => {
    const row = getDbService().updatePayment(data.id, {
        member_id: data.memberId,
        amount_cents: data.amountCents,
        currency: data.currency,
        date: data.date,
        note: data.note,
    });
    if (!row) throw new Error(`Payment ${data.id} not found.`);
    return dbPaymentToRecord(row);
});

ipcMain.handle('payments:delete', (_event, id: string) => {
    getDbService().deletePayment(id);
    return { ok: true };
});

// ── Attachments IPC handlers ──────────────────────────────────────────────────

function dbAttachmentToRecord(a: import('./storage/SqliteDbService.js').DbAttachment): AttachmentRecord {
    return {
        id: a.id,
        originalName: a.original_name,
        storedRelPath: a.stored_rel_path,
        mimeType: a.mime_type,
        sizeBytes: a.size_bytes,
        sha256: a.sha256,
        paymentId: a.payment_id,
        memberId: a.member_id,
        createdAt: a.created_at,
    };
}

ipcMain.handle(
    'attachments:add',
    async (
        _event,
        {
            sourcePath,
            originalName,
            mimeType,
            paymentId,
            memberId,
        }: {
            sourcePath: string;
            originalName: string;
            mimeType: string;
            paymentId?: string;
            memberId?: string;
        }
    ) => {
        const db = getDbService();
        const attachSvc = getAttachmentService();
        const uuid = crypto.randomUUID();

        const { storedRelPath, sha256, sizeBytes } = await attachSvc.writeAttachment(
            sourcePath,
            originalName,
            uuid
        );

        // Insert metadata inside a transaction; roll back and remove the file
        // if the DB insert fails.
        let inserted: AttachmentRecord;
        try {
            const row = db.transaction(() =>
                db.insertAttachment({
                    id: uuid,
                    original_name: originalName,
                    stored_rel_path: storedRelPath,
                    mime_type: mimeType,
                    size_bytes: sizeBytes,
                    sha256,
                    payment_id: paymentId ?? null,
                    member_id: memberId ?? null,
                    created_at: new Date().toISOString(),
                })
            );
            inserted = dbAttachmentToRecord(row);
        } catch (err) {
            // Remove the already-written file to keep the store consistent.
            try {
                const absPath = attachSvc.resolvePath(currentDataDir!, storedRelPath);
                try { fs.unlinkSync(absPath); } catch (cleanupErr) {
                    console.error('Failed to clean up attachment file after DB insert failure:', cleanupErr);
                }
            } catch (cleanupErr) {
                console.error('Failed to resolve attachment path for cleanup:', cleanupErr);
            }
            throw err;
        }

        return inserted;
    }
);

ipcMain.handle('attachments:list', (_event, filter?: { paymentId?: string; memberId?: string }) => {
    const rows = getDbService().listAttachments(
        filter
            ? { payment_id: filter.paymentId, member_id: filter.memberId }
            : undefined
    );
    return rows.map(dbAttachmentToRecord);
});

ipcMain.handle('attachments:open', async (_event, id: string) => {
    const row = getDbService().getAttachment(id);
    if (!row) throw new Error(`Attachment ${id} not found.`);
    const absPath = getAttachmentService().resolvePath(currentDataDir!, row.stored_rel_path);
    await shell.openPath(absPath);
    return { ok: true };
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
        stopLockHeartbeat();
        lockService?.release();
        dbService?.close();
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

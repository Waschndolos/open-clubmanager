const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('userpreference', {
  get(userId: string, key: string) {
    return ipcRenderer.sendSync('userpreference-get', userId, key);
  },
    
  set(userId: string, key: string, value: unknown) {
    ipcRenderer.send('userpreference-set', userId, key, value);
  }
});

contextBridge.exposeInMainWorld('apppreference', {
  get(key: string) {
    return ipcRenderer.sendSync('apppreference-get', key);
  },
    
  set(key: string, value: unknown) {
    ipcRenderer.send('apppreference-set', key, value);
  }
});

contextBridge.exposeInMainWorld('electronDialog', {
  selectDbDirectory(): Promise<string | null> {
    return ipcRenderer.invoke('dialog:select-db-directory');
  },
  selectDbFile(): Promise<string | null> {
    return ipcRenderer.invoke('dialog:select-db-file');
  },
  getStoredDbPath(): string | null {
    return ipcRenderer.sendSync('db-path-get');
  },
  storeDbPath(dbPath: string): void {
    ipcRenderer.send('db-path-set', dbPath);
  },
});
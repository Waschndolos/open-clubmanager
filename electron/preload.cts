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
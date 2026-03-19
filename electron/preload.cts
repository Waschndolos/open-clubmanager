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

contextBridge.exposeInMainWorld('api', {
  club: {
    selectFolder: () => ipcRenderer.invoke('club:selectFolder'),
    getFolder: () => ipcRenderer.invoke('club:getFolder'),
    initFolder: () => ipcRenderer.invoke('club:initFolder'),
  },
  members: {
    list: () => ipcRenderer.invoke('members:list'),
    get: (id: number) => ipcRenderer.invoke('members:get', id),
    create: (data: unknown) => ipcRenderer.invoke('members:create', data),
    update: (id: number, patch: unknown) => ipcRenderer.invoke('members:update', { id, patch }),
    delete: (ids: number[]) => ipcRenderer.invoke('members:delete', ids),
    lock: (id: number, owner: string) => ipcRenderer.invoke('members:lock', { id, owner }),
    unlock: (id: number, owner: string) => ipcRenderer.invoke('members:unlock', { id, owner }),
    getLock: (id: number) => ipcRenderer.invoke('members:getLock', id),
  },
  roles: {
    list: () => ipcRenderer.invoke('role:list'),
    create: (data: unknown) => ipcRenderer.invoke('role:create', data),
    update: (data: unknown) => ipcRenderer.invoke('role:update', data),
    delete: (id: number) => ipcRenderer.invoke('role:delete', id),
  },
  groups: {
    list: () => ipcRenderer.invoke('group:list'),
    create: (data: unknown) => ipcRenderer.invoke('group:create', data),
    update: (data: unknown) => ipcRenderer.invoke('group:update', data),
    delete: (id: number) => ipcRenderer.invoke('group:delete', id),
  },
  sections: {
    list: () => ipcRenderer.invoke('section:list'),
    create: (data: unknown) => ipcRenderer.invoke('section:create', data),
    update: (data: unknown) => ipcRenderer.invoke('section:update', data),
    delete: (id: number) => ipcRenderer.invoke('section:delete', id),
  },
  storage: {
    getStatus: () => ipcRenderer.invoke('storage:getStatus'),
    requestEditMode: () => ipcRenderer.invoke('storage:requestEditMode'),
    releaseEditMode: () => ipcRenderer.invoke('storage:releaseEditMode'),
    exportBackup: () => ipcRenderer.invoke('storage:exportBackup'),
  },
  payments: {
    list: () => ipcRenderer.invoke('payments:list'),
    create: (data: unknown) => ipcRenderer.invoke('payments:create', data),
    update: (data: unknown) => ipcRenderer.invoke('payments:update', data),
    delete: (id: string) => ipcRenderer.invoke('payments:delete', id),
  },
  attachments: {
    add: (data: unknown) => ipcRenderer.invoke('attachments:add', data),
    list: (filter?: unknown) => ipcRenderer.invoke('attachments:list', filter),
    open: (id: string) => ipcRenderer.invoke('attachments:open', id),
  },
});
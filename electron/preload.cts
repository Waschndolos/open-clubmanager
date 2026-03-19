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
  finance: {
    listTransactions: () => ipcRenderer.invoke('finance:listTransactions'),
    createTransaction: (data: unknown) => ipcRenderer.invoke('finance:createTransaction', data),
    updateTransaction: (data: unknown) => ipcRenderer.invoke('finance:updateTransaction', data),
    deleteTransaction: (id: number) => ipcRenderer.invoke('finance:deleteTransaction', id),
    listMemberFees: () => ipcRenderer.invoke('finance:listMemberFees'),
    createMemberFee: (data: unknown) => ipcRenderer.invoke('finance:createMemberFee', data),
    updateMemberFee: (data: unknown) => ipcRenderer.invoke('finance:updateMemberFee', data),
    deleteMemberFee: (id: number) => ipcRenderer.invoke('finance:deleteMemberFee', id),
  },
  history: {
    list: () => ipcRenderer.invoke('history:list'),
  },
});
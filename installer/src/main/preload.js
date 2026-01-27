const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('mycelicmemory', {
  checkDependencies: () => ipcRenderer.invoke('check-dependencies'),
  installMyclicMemory: () => ipcRenderer.invoke('install-mycelicmemory'),
  openExternal: (url) => ipcRenderer.invoke('open-external', url),
  launchMyclicMemory: () => ipcRenderer.invoke('launch-mycelicmemory'),
  onInstallProgress: (callback) => {
    ipcRenderer.on('install-progress', (event, progress) => callback(progress));
  }
});

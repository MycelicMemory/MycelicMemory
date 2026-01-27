const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('ultrathink', {
  checkDependencies: () => ipcRenderer.invoke('check-dependencies'),
  installUltrathink: () => ipcRenderer.invoke('install-ultrathink'),
  openExternal: (url) => ipcRenderer.invoke('open-external', url),
  launchUltrathink: () => ipcRenderer.invoke('launch-ultrathink'),
  onInstallProgress: (callback) => {
    ipcRenderer.on('install-progress', (event, progress) => callback(progress));
  }
});

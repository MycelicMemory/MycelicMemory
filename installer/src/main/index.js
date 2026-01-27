const { app, BrowserWindow, ipcMain, shell, protocol, net } = require('electron');
const path = require('path');
const fs = require('fs');
const { checkDependencies, installMyclicMemory } = require('./dependencies');

let mainWindow;
const isDashboardMode = process.argv.includes('--dashboard');

function createWindow() {
  if (isDashboardMode) {
    createDashboardWindow();
  } else {
    createInstallerWindow();
  }
}

function createInstallerWindow() {
  mainWindow = new BrowserWindow({
    width: 600,
    height: 500,
    resizable: false,
    frame: true,
    titleBarStyle: 'hiddenInset',
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, 'preload.js')
    }
  });

  mainWindow.loadFile(path.join(__dirname, '../renderer/index.html'));

  if (process.argv.includes('--dev')) {
    mainWindow.webContents.openDevTools();
  }
}

function createDashboardWindow() {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    minWidth: 900,
    minHeight: 600,
    frame: true,
    titleBarStyle: 'hiddenInset',
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true
    }
  });

  // Get dashboard path - in packaged app it's in resources, in dev it's in parent dashboard folder
  let dashboardPath;
  if (app.isPackaged) {
    dashboardPath = path.join(process.resourcesPath, 'dashboard');
  } else {
    dashboardPath = path.join(__dirname, '../../../dashboard/dist');
  }

  // Check if dashboard exists
  const indexPath = path.join(dashboardPath, 'index.html');
  if (fs.existsSync(indexPath)) {
    mainWindow.loadFile(indexPath);
  } else {
    // Fallback: show error message
    mainWindow.loadURL(`data:text/html,
      <html>
        <body style="font-family: system-ui; padding: 40px; background: #1e293b; color: #f1f5f9;">
          <h1>Dashboard Not Found</h1>
          <p>Build the dashboard first: <code>cd dashboard && npm run build</code></p>
        </body>
      </html>
    `);
  }

  if (process.argv.includes('--dev')) {
    mainWindow.webContents.openDevTools();
  }
}

app.whenReady().then(createWindow);

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

// IPC Handlers
ipcMain.handle('check-dependencies', async () => {
  return await checkDependencies();
});

ipcMain.handle('install-mycelicmemory', async (event) => {
  return await installMyclicMemory((progress) => {
    event.sender.send('install-progress', progress);
  });
});

ipcMain.handle('open-external', async (event, url) => {
  await shell.openExternal(url);
  return true;
});

ipcMain.handle('launch-mycelicmemory', async () => {
  const { exec } = require('child_process');
  exec('mycelicmemory doctor', (error, stdout, stderr) => {
    if (error) {
      console.error('Launch error:', error);
    }
  });
  return true;
});

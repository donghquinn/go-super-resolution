const { app, BrowserWindow, dialog, ipcMain } = require('electron');
const { execFile } = require('node:child_process');
const fs = require('node:fs/promises');
const path = require('node:path');
const { pathToFileURL } = require('node:url');

const IMAGE_EXTENSIONS = ['jpg', 'jpeg', 'png', 'bmp', 'tiff'];
const MODEL_EXTENSIONS = ['onnx', 'pb', 'tflite'];

let mainWindow;

function backendPath() {
  if (process.env.SR_BACKEND) return process.env.SR_BACKEND;
  const exe = process.platform === 'win32' ? 'sr-backend.exe' : 'sr-backend';
  const base = app.isPackaged ? process.resourcesPath : path.join(__dirname, '..');
  return path.join(base, 'bin', exe);
}

// Same naming scheme as the old Fyne GUI: <name>_sr_<engine>_<scale><ext> next to the input.
function outputPathFor(inputPath, engine, scaleLabel) {
  const ext = path.extname(inputPath);
  const name = path.basename(inputPath, ext);
  return path.join(path.dirname(inputPath), `${name}_sr_${engine}_${scaleLabel}${ext}`);
}

function runBackend(args) {
  return new Promise((resolve, reject) => {
    execFile(backendPath(), args, { maxBuffer: 1024 * 1024 }, (err, stdout, stderr) => {
      const line = stdout.trim().split('\n').pop();
      let result;
      try {
        result = JSON.parse(line);
      } catch {
        // Backend crashed or couldn't start (e.g. missing binary / OpenCV libs).
        return reject(new Error((stderr || err?.message || 'Backend produced no output').trim()));
      }
      if (!result.ok) return reject(new Error(result.error));
      resolve(result);
    });
  });
}

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    minWidth: 900,
    minHeight: 600,
    title: 'Super Resolution - AI Image Upscaler',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  });
  mainWindow.loadFile(path.join(__dirname, 'renderer', 'index.html'));
}

ipcMain.handle('file-url', (_e, filePath) => pathToFileURL(filePath).href);

ipcMain.handle('open-image', async () => {
  const { canceled, filePaths } = await dialog.showOpenDialog(mainWindow, {
    properties: ['openFile'],
    filters: [{ name: 'Images', extensions: IMAGE_EXTENSIONS }],
  });
  return canceled ? null : filePaths[0];
});

ipcMain.handle('open-model', async () => {
  const { canceled, filePaths } = await dialog.showOpenDialog(mainWindow, {
    properties: ['openFile'],
    filters: [{ name: 'Models', extensions: MODEL_EXTENSIONS }],
  });
  return canceled ? null : filePaths[0];
});

ipcMain.handle('save-result', async (_e, resultPath) => {
  const { canceled, filePath } = await dialog.showSaveDialog(mainWindow, {
    defaultPath: `super_resolution_result${path.extname(resultPath)}`,
  });
  if (canceled || !filePath) return null;
  await fs.copyFile(resultPath, filePath);
  return filePath;
});

ipcMain.handle('process', async (_e, opts) => {
  const scaleLabel = opts.scale === 'custom' ? 'Custom' : `${opts.scale}x`;
  const outputPath = outputPathFor(opts.inputPath, opts.engine, scaleLabel);
  const custom = opts.scale === 'custom';
  return runBackend([
    '-input', opts.inputPath,
    '-output', outputPath,
    '-model', opts.modelPath,
    '-mode', opts.mode,
    '-engine', opts.engine,
    '-scale', String(custom ? 1 : opts.scale),
    '-width', String(custom ? opts.width || 0 : 0),
    '-height', String(custom ? opts.height || 0 : 0),
  ]);
});

app.whenReady().then(() => {
  createWindow();
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});

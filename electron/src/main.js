const { app, BrowserWindow, dialog, ipcMain, shell } = require('electron');
const { spawn } = require('node:child_process');
const fs = require('node:fs/promises');
const path = require('node:path');
const { pathToFileURL } = require('node:url');
const { ModelStore } = require('./models');

const IMAGE_EXTENSIONS = ['jpg', 'jpeg', 'png', 'bmp', 'tiff'];
const MODEL_EXTENSIONS = ['onnx', 'pb', 'tflite'];

let mainWindow;

function backendPath() {
  if (process.env.SR_BACKEND) return process.env.SR_BACKEND;
  const exe = process.platform === 'win32' ? 'sr-backend.exe' : 'sr-backend';
  const base = app.isPackaged ? process.resourcesPath : path.join(__dirname, '..');
  return path.join(base, 'bin', exe);
}

function modelsDir() {
  return app.isPackaged ? path.join(app.getPath('userData'), 'models') : path.join(__dirname, '..', '..', 'models');
}

let models; // ModelStore, created once the app is ready

// Same naming scheme as the old Fyne GUI: <name>_sr_<engine or model>_<scale><ext> next to the input.
function outputPathFor(inputPath, tag, scaleLabel) {
  const ext = path.extname(inputPath);
  const name = path.basename(inputPath, ext);
  return path.join(path.dirname(inputPath), `${name}_sr_${tag}_${scaleLabel}${ext}`);
}

// Runs the backend; it prints "progress <0-1>" lines on stderr and one JSON result line on stdout.
function runBackend(args, onProgress) {
  return new Promise((resolve, reject) => {
    const child = spawn(backendPath(), args);
    let stdout = '';
    let stderr = '';
    let pending = '';
    child.stdout.setEncoding('utf8').on('data', (d) => { stdout += d; });
    child.stderr.setEncoding('utf8').on('data', (d) => {
      const lines = (pending + d).split('\n');
      pending = lines.pop();
      for (const line of lines) {
        const m = /^progress ([\d.]+)$/.exec(line);
        if (m) onProgress(Number(m[1]));
        else stderr += `${line}\n`;
      }
    });
    child.on('error', (err) => reject(new Error(`Could not start backend (${backendPath()}): ${err.message}`)));
    child.on('close', (code, signal) => {
      let result;
      try {
        result = JSON.parse(stdout.trim().split('\n').pop());
      } catch {
        // Backend crashed or couldn't start (e.g. missing OpenCV / ONNX Runtime libs).
        return reject(new Error((stderr + pending).trim() || `Backend exited (${signal || code}) without a result`));
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

ipcMain.handle('list-models', async () => ({ models: await models.list(), dir: models.dir }));
ipcMain.handle('download-model', (_e, id) => models.ensure(id).then(() => undefined));
ipcMain.handle('cancel-download', (_e, id) => models.cancel(id));
ipcMain.handle('delete-model', (_e, id) => models.remove(id));
ipcMain.handle('add-model-url', (_e, url) => models.addFromUrl(url));
ipcMain.handle('open-models-folder', async () => {
  await fs.mkdir(models.dir, { recursive: true });
  return shell.openPath(models.dir);
});

// opts.modelId selects a managed model; otherwise opts.modelPath is a user-chosen file.
// Upscaling progress is sent as 'progress'; model downloads report via 'model-download'.
ipcMain.handle('process', async (e, opts) => {
  const model = opts.modelId ? await models.find(opts.modelId) : null;
  if (opts.modelId && !model) throw new Error('The selected model is no longer available');
  const modelPath = model ? await models.ensure(model.id) : opts.modelPath;

  const custom = opts.scale === 'custom';
  const scaleLabel = custom ? 'Custom' : `${opts.scale}x`;
  const tag = model ? model.id.replace(/^file:/, '').replace(/\.onnx$/i, '') : opts.engine;
  const outputPath = outputPathFor(opts.inputPath, tag, scaleLabel);
  const result = await runBackend([
    '-input', opts.inputPath,
    '-output', outputPath,
    '-model', modelPath,
    '-mode', opts.mode,
    '-engine', opts.engine,
    '-scale', String(custom ? 1 : opts.scale),
    '-width', String(custom ? opts.width || 0 : 0),
    '-height', String(custom ? opts.height || 0 : 0),
  ], (fraction) => e.sender.send('progress', { fraction }));
  return result;
});

// Which execution devices (CUDA, CoreML, CPU) this machine can use; probed once.
let devices;
ipcMain.handle('list-devices', () => {
  devices ??= runBackend(['-devices'], () => {}).then((r) => r.devices).catch((err) => {
    devices = undefined; // retry next time, e.g. after ONNX Runtime is installed
    throw err;
  });
  return devices;
});

app.whenReady().then(() => {
  models = new ModelStore(modelsDir(), {
    inspect: async (file) => (await runBackend(['-inspect', '-model', file], () => {})).model,
    onEvent: (event) => {
      for (const win of BrowserWindow.getAllWindows()) win.webContents.send('model-download', event);
    },
  });
  createWindow();
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});

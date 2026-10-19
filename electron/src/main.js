const { app, BrowserWindow, dialog, ipcMain } = require('electron');
const { spawn } = require('node:child_process');
const { once } = require('node:events');
const fsSync = require('node:fs');
const fs = require('node:fs/promises');
const path = require('node:path');
const { pathToFileURL } = require('node:url');

const IMAGE_EXTENSIONS = ['jpg', 'jpeg', 'png', 'bmp', 'tiff'];
const MODEL_EXTENSIONS = ['onnx', 'pb', 'tflite'];

// Built-in ONNX models, downloaded on first use. 4x-UltraSharpV2 by Kim2091 (CC-BY-NC-SA-4.0).
// coreml: whether CoreML ("mps" mode) speeds the model up; DAT2 is slower on CoreML than on CPU.
const MODELS = [
  {
    id: 'ultrasharp-v2',
    label: 'UltraSharpV2 (best quality)',
    file: '4x-UltraSharpV2_fp32_op17.onnx',
    url: 'https://huggingface.co/Kim2091/UltraSharpV2/resolve/main/4x-UltraSharpV2_fp32_op17.onnx',
    sizeMB: 52,
    coreml: false,
  },
  {
    id: 'ultrasharp-v2-lite',
    label: 'UltraSharpV2 Lite (faster)',
    file: '4x-UltraSharpV2_Lite_fp32_op17.onnx',
    url: 'https://huggingface.co/Kim2091/UltraSharpV2/resolve/main/4x-UltraSharpV2_Lite_fp32_op17.onnx',
    sizeMB: 30,
    coreml: true,
  },
];

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

const modelPath = (model) => path.join(modelsDir(), model.file);

async function downloadModel(model, onProgress) {
  const dest = modelPath(model);
  const part = `${dest}.part`;
  await fs.mkdir(path.dirname(dest), { recursive: true });
  try {
    const res = await fetch(model.url);
    if (!res.ok) throw new Error(`Model download failed: HTTP ${res.status}`);
    const total = Number(res.headers.get('content-length')) || 0;
    const out = fsSync.createWriteStream(part);
    let received = 0;
    let reported = -1;
    for await (const chunk of res.body) {
      received += chunk.length;
      if (!out.write(chunk)) await once(out, 'drain');
      const percent = total ? Math.floor((received / total) * 100) : 0;
      if (percent !== reported) onProgress(percent / 100);
      reported = percent;
    }
    await new Promise((resolve, reject) => out.end((err) => (err ? reject(err) : resolve())));
    if (total && received !== total) throw new Error('Model download was incomplete');
    await fs.rename(part, dest);
  } catch (err) {
    await fs.rm(part, { force: true });
    throw err;
  }
}

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

ipcMain.handle('list-models', () =>
  MODELS.map((m) => ({ id: m.id, label: m.label, sizeMB: m.sizeMB, downloaded: fsSync.existsSync(modelPath(m)) })));

// opts.modelId selects a built-in model; otherwise opts.modelPath is a user-chosen file.
// Progress is reported to the renderer as { phase: 'download' | 'process', fraction, ... }.
ipcMain.handle('process', async (e, opts) => {
  const report = (info) => e.sender.send('progress', info);
  const model = MODELS.find((m) => m.id === opts.modelId);
  let { mode } = opts;
  const notes = [];

  if (model) {
    if (!fsSync.existsSync(modelPath(model))) {
      await downloadModel(model, (fraction) =>
        report({ phase: 'download', fraction, label: model.label, sizeMB: model.sizeMB }));
    }
    if (mode === 'mps' && !model.coreml) {
      mode = 'cpu';
      notes.push('ran on CPU: this model is slower with CoreML');
    }
  }

  const custom = opts.scale === 'custom';
  const scaleLabel = custom ? 'Custom' : `${opts.scale}x`;
  const outputPath = outputPathFor(opts.inputPath, model ? model.id : opts.engine, scaleLabel);
  const result = await runBackend([
    '-input', opts.inputPath,
    '-output', outputPath,
    '-model', model ? modelPath(model) : opts.modelPath,
    '-mode', mode,
    '-engine', opts.engine,
    '-scale', String(custom ? 1 : opts.scale),
    '-width', String(custom ? opts.width || 0 : 0),
    '-height', String(custom ? opts.height || 0 : 0),
  ], (fraction) => report({ phase: 'process', fraction }));
  return { ...result, notes };
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

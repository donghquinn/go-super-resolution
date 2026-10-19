const IMAGE_EXT = /\.(jpe?g|png|bmp|tiff?)$/i;
const DEVICE_NAMES = { cuda: 'NVIDIA GPU (CUDA)', coreml: 'Apple GPU (CoreML)', cpu: 'CPU' };
const GPU_PREF_KEY = 'useGpu';
// The engine follows from the model; main.js derives it the same way when processing.
const ENGINES = {
  onnx: 'ONNX Runtime',
  opencv: 'OpenCV (resize only)',
  tensorflow: 'TensorFlow',
  tflite: 'TensorFlow Lite',
};
const ENGINE_BY_EXT = { onnx: 'onnx', pb: 'tensorflow', tflite: 'tflite' };

const $ = (id) => document.getElementById(id);
const els = {
  dropZone: $('drop-zone'),
  engine: $('engine'),
  useGpu: $('use-gpu'),
  deviceHint: $('device-hint'),
  scale: $('scale'),
  customSize: $('custom-size'),
  width: $('width'),
  height: $('height'),
  model: $('model'),
  modelSelect: $('model-select'),
  customModel: $('custom-model'),
  modelHint: $('model-hint'),
  manageModels: $('manage-models'),
  modelsDialog: $('models-dialog'),
  modelList: $('model-list'),
  urlForm: $('url-form'),
  modelUrl: $('model-url'),
  urlSubmit: $('url-submit'),
  urlStatus: $('url-status'),
  modelsDir: $('models-dir'),
  process: $('process'),
  save: $('save'),
  clear: $('clear'),
  originalImg: $('original-img'),
  resultImg: $('result-img'),
  originalSize: $('original-size'),
  resultSize: $('result-size'),
  status: $('status'),
  zoomOriginal: $('zoom-original'),
  zoomResult: $('zoom-result'),
  downloadOriginal: $('download-original'),
  downloadResult: $('download-result'),
  progress: $('progress'),
};

const state = {
  imagePath: null,
  resultPath: null,
  processing: false,
  models: [],
  activeModelId: null,
  gpus: null, // usable GPU devices, once detected
};

const basename = (p) => p.split(/[\\/]/).pop();
const setStatus = (text) => { els.status.textContent = text; };
const errorText = (err) => err.message.replace(/^Error invoking remote method '[^']+': (Error: )?/, '');

// Cache-bust so re-processing to the same output path shows the new file.
async function showImage(img, filePath) {
  const url = await window.sr.fileUrl(filePath);
  await new Promise((resolve, reject) => {
    img.onload = resolve;
    img.onerror = () => reject(new Error(`Failed to load image: ${basename(filePath)}`));
    img.src = `${url}?t=${Date.now()}`;
    img.hidden = false;
  });
  img.nextElementSibling.hidden = true;
}

// Zoom/Download buttons follow which images are loaded.
function updateImageActions() {
  els.zoomOriginal.disabled = els.downloadOriginal.disabled = !state.imagePath;
  els.zoomResult.disabled = els.downloadResult.disabled = !state.resultPath;
  els.save.disabled = !state.resultPath;
}

const viewer = new ImageViewer($('viewer'), { onDownload: (p) => downloadImage(p) });

function openViewer(view) {
  if (!state.imagePath || (view === 'result' && !state.resultPath)) return;
  const source = (img, filePath) => filePath && {
    path: filePath, url: img.src, width: img.naturalWidth, height: img.naturalHeight,
  };
  viewer.open({
    original: source(els.originalImg, state.imagePath),
    result: source(els.resultImg, state.resultPath),
  }, view);
}

async function downloadImage(filePath) {
  try {
    const saved = await window.sr.saveImage(filePath);
    if (saved) setStatus(`Saved: ${saved}`);
  } catch (err) {
    setStatus(`Error: ${errorText(err)}`);
  }
}

function resetImage(img, sizeEl) {
  img.removeAttribute('src');
  img.hidden = true;
  img.nextElementSibling.hidden = false;
  sizeEl.textContent = '';
}

async function loadImage(filePath) {
  if (!IMAGE_EXT.test(filePath)) {
    setStatus(`Error: unsupported file type: ${basename(filePath)}`);
    return;
  }
  try {
    await showImage(els.originalImg, filePath);
  } catch (err) {
    setStatus(`Error: ${err.message}`);
    return;
  }
  state.imagePath = filePath;
  state.resultPath = null;
  resetImage(els.resultImg, els.resultSize);
  updateImageActions();
  const { naturalWidth: w, naturalHeight: h } = els.originalImg;
  els.originalSize.textContent = `(${w}×${h})`;
  setStatus(`Loaded: ${basename(filePath)} (${w}x${h})`);
}

// Detects usable GPUs once at startup. GPU use is on by default; the checkbox lets users opt out
// (remembered per machine), which runs everything on CPU.
async function detectGpus() {
  try {
    const devices = await window.sr.listDevices();
    state.gpus = devices.filter((d) => d.available && d.id !== 'cpu');
  } catch (err) {
    state.gpus = [];
    els.deviceHint.title = errorText(err);
  }
  updateGpuUI();
}

function updateGpuUI() {
  if (!state.gpus) return;
  const names = state.gpus.map((d) => DEVICE_NAMES[d.id] || d.name).join(', ');
  els.useGpu.disabled = state.gpus.length === 0;
  if (!state.gpus.length) {
    els.deviceHint.textContent = 'No usable GPU found: runs on CPU';
  } else if (els.useGpu.checked) {
    els.deviceHint.textContent = `Detected ${names}`;
  } else {
    els.deviceHint.textContent = `GPU off: runs on CPU (${names} available)`;
  }
}

function loadGpuPreference() {
  try {
    els.useGpu.checked = localStorage.getItem(GPU_PREF_KEY) !== 'false';
  } catch {
    // storage unavailable: keep the default (on)
  }
}

function saveGpuPreference() {
  try {
    localStorage.setItem(GPU_PREF_KEY, String(els.useGpu.checked));
  } catch {
    // not critical
  }
}

// (Re)builds the model menu: downloaded models first, then ones that download on first use.
async function refreshModels() {
  const firstLoad = state.models.length === 0;
  const selected = els.modelSelect.value;
  const { models, dir } = await window.sr.listModels();
  state.models = models;
  els.modelsDir.textContent = dir;

  for (const node of [...els.modelSelect.children]) if (node.label !== 'Other') node.remove();
  const group = (label, items) => {
    const g = document.createElement('optgroup');
    g.label = label;
    g.append(...items.map((m) => new Option(m.source === 'catalog' ? `${m.label} (${m.scale}x)` : m.label, m.id)));
    return g;
  };
  const ready = models.filter((m) => m.downloaded);
  const available = models.filter((m) => !m.downloaded && m.source === 'catalog');
  els.modelSelect.prepend(
    ...(ready.length ? [group('Downloaded', ready)] : []),
    ...(available.length ? [group('Download on first use', available)] : []),
  );

  const exists = (id) => models.some((m) => m.id === id && (m.downloaded || m.source === 'catalog'));
  const keep = !firstLoad && (selected === 'custom' || selected === 'resize' || exists(selected));
  els.modelSelect.value = keep ? selected : models[0].id;
  updateModelUI();
  if (els.modelsDialog.open) renderModelList();
}

// Engine for the current model choice: managed models are ONNX, "Basic resize" is OpenCV, and a
// custom file's engine comes from its extension (null until a supported file is chosen).
function currentEngine() {
  const choice = els.modelSelect.value;
  if (choice === 'resize') return 'opencv';
  if (choice !== 'custom') return 'onnx';
  const ext = els.model.value.trim().split('.').pop().toLowerCase();
  return ENGINE_BY_EXT[ext] || null;
}

function updateModelUI() {
  const choice = els.modelSelect.value;
  const model = state.models.find((m) => m.id === choice);
  const engine = currentEngine();
  els.customModel.hidden = choice !== 'custom';
  els.engine.value = engine ? ENGINES[engine] : '—';

  if (model) {
    els.modelHint.textContent = modelSummary(model);
  } else if (choice === 'resize') {
    els.modelHint.textContent = 'Plain bicubic resize, no AI model';
  } else if (!engine) {
    els.modelHint.textContent = 'Choose a .onnx, .pb or .tflite model file';
  } else if (engine === 'onnx') {
    els.modelHint.textContent = '';
  } else {
    els.modelHint.textContent = `${ENGINES[engine]} inference isn't implemented yet: the image is resized without AI`;
  }
}

function modelSummary(m) {
  if (m.source === 'user') {
    const details = m.scale ? `${m.scale}x · ${m.precision}` : 'not verified';
    return `${details} · ${m.url ? `from ${new URL(m.url).hostname}` : 'added manually'}`;
  }
  const status = m.downloaded ? 'Downloaded' : `Downloads ${m.sizeMB} MB on first use`;
  return `${m.arch} · ${m.license}${m.license.includes('NC') ? ' (non-commercial)' : ''} · ${status}`;
}

function button(label, onClick, className) {
  const b = document.createElement('button');
  b.type = 'button';
  b.textContent = label;
  if (className) b.className = className;
  b.addEventListener('click', onClick);
  return b;
}

function renderModelList() {
  els.modelList.replaceChildren(...state.models.map((m) => {
    const li = document.createElement('li');
    li.className = 'model-item';
    li.dataset.id = m.id;

    const info = document.createElement('div');
    info.className = 'model-info';
    const title = document.createElement('strong');
    title.textContent = m.label;
    const desc = document.createElement('p');
    desc.className = 'hint';
    desc.textContent = m.source === 'catalog' ? m.description : 'Added by you';
    const meta = document.createElement('p');
    meta.className = 'hint';
    meta.textContent = m.source === 'catalog'
      ? `${m.scale}x · ${m.arch} · ${m.sizeMB} MB · ${m.license} · by ${m.author}`
      : [m.scale && `${m.scale}x`, m.precision, m.sizeMB && `${m.sizeMB} MB`, m.url && new URL(m.url).hostname]
        .filter(Boolean).join(' · ');
    info.append(title, desc, meta);

    const actions = document.createElement('div');
    actions.className = 'model-actions';
    if (m.downloading) {
      const bar = document.createElement('progress');
      if (m.progress != null) bar.value = m.progress;
      actions.append(bar, button('Cancel', () => window.sr.cancelDownload(m.id)));
    } else if (m.downloaded) {
      actions.append(
        button('Use', () => selectModel(m.id), 'primary'),
        button('Delete', () => deleteModel(m)),
      );
    } else {
      actions.append(button('Download', () => downloadModel(m.id)));
    }
    li.append(info, actions);
    return li;
  }));
}

function selectModel(id) {
  els.modelSelect.value = id;
  updateModelUI();
  els.modelsDialog.close();
}

async function downloadModel(id) {
  const pending = window.sr.downloadModel(id);
  await refreshModels(); // shows the row as downloading
  try {
    await pending;
  } catch (err) {
    if (!/canceled/.test(err.message)) els.urlStatus.textContent = `Download failed: ${errorText(err)}`;
  }
}

async function deleteModel(m) {
  if (!window.confirm(`Delete ${m.label}?${m.source === 'catalog' ? ' You can download it again later.' : ''}`)) return;
  try {
    await window.sr.deleteModel(m.id);
  } catch (err) {
    els.urlStatus.textContent = `Could not delete: ${errorText(err)}`;
  }
  await refreshModels();
}

async function addModelFromUrl(e) {
  e.preventDefault();
  const url = els.modelUrl.value;
  els.urlSubmit.disabled = true;
  els.urlStatus.textContent = 'Downloading...';
  try {
    const pending = window.sr.addModelUrl(url);
    setTimeout(refreshModels, 300); // show the new row once the download has started
    const added = await pending;
    els.modelUrl.value = '';
    els.urlStatus.textContent = added.scale
      ? `Added: verified ${added.scale}x ${added.precision} model.`
      : 'Added.';
    await refreshModels();
  } catch (err) {
    els.urlStatus.textContent = `Could not add model: ${errorText(err)}`;
    await refreshModels();
  } finally {
    els.urlSubmit.disabled = false;
  }
}

// Download progress, for both the manager dialog and a download started by Process Image.
function onModelDownload(event) {
  if (event.done) {
    if (event.id.startsWith('file:') && !event.error) els.urlStatus.textContent = 'Verifying model...';
    refreshModels();
    return;
  }
  const model = state.models.find((m) => m.id === event.id);
  if (model) model.progress = event.fraction;
  const bar = els.modelList.querySelector(`[data-id="${CSS.escape(event.id)}"] progress`);
  if (bar && event.fraction != null) bar.value = event.fraction;
  if (state.processing && event.id === state.activeModelId) {
    showProgress(event.fraction, `Downloading ${event.label}...`);
  }
}

function showProgress(fraction, label) {
  if (fraction == null) {
    els.progress.removeAttribute('value');
    setStatus(label);
    return;
  }
  els.progress.value = fraction;
  setStatus(`${label} ${Math.round(fraction * 100)}%`);
}

async function processImage() {
  if (!state.imagePath) {
    setStatus('Please select an image first');
    return;
  }
  if (state.processing) return;

  state.processing = true;
  els.process.disabled = true;
  els.progress.removeAttribute('value'); // indeterminate until the first progress report
  els.progress.hidden = false;
  setStatus('Processing...');

  const choice = els.modelSelect.value;
  const modelId = choice === 'custom' || choice === 'resize' ? null : choice;
  state.activeModelId = modelId;
  try {
    const result = await window.sr.process({
      inputPath: state.imagePath,
      modelId,
      resize: choice === 'resize',
      modelPath: els.model.value.trim(),
      mode: els.useGpu.checked ? 'auto' : 'cpu', // auto: fastest usable GPU, else CPU
      scale: els.scale.value === 'custom' ? 'custom' : Number(els.scale.value),
      width: Number(els.width.value) || 0,
      height: Number(els.height.value) || 0,
    });
    await showImage(els.resultImg, result.outputPath);
    state.resultPath = result.outputPath;
    els.resultSize.textContent = `(${els.resultImg.naturalWidth}×${els.resultImg.naturalHeight})`;
    updateImageActions();
    const device = result.device ? ` on ${DEVICE_NAMES[result.device] || result.device}` : '';
    const warnings = result.warnings?.length ? ` (${result.warnings.join('; ')})` : '';
    setStatus(`Processing completed successfully${device}! Saved to ${basename(result.outputPath)}${warnings}`);
  } catch (err) {
    setStatus(`Processing failed: ${errorText(err)}`);
  } finally {
    state.processing = false;
    els.process.disabled = false;
    els.progress.hidden = true;
  }
}

function saveResult() {
  if (state.resultPath) downloadImage(state.resultPath);
}

function clearAll() {
  state.imagePath = null;
  state.resultPath = null;
  updateImageActions();
  resetImage(els.originalImg, els.originalSize);
  resetImage(els.resultImg, els.resultSize);
  setStatus('Ready - Select an image to get started');
}

$('browse-image').addEventListener('click', async () => {
  const filePath = await window.sr.openImage();
  if (filePath) loadImage(filePath);
});

$('browse-model').addEventListener('click', async () => {
  const filePath = await window.sr.openModel();
  if (filePath) {
    els.model.value = filePath;
    updateModelUI();
  }
});

els.scale.addEventListener('change', () => {
  els.customSize.hidden = els.scale.value !== 'custom';
});

els.modelSelect.addEventListener('change', updateModelUI);
els.model.addEventListener('input', updateModelUI);
els.process.addEventListener('click', processImage);
els.save.addEventListener('click', saveResult);
els.zoomOriginal.addEventListener('click', () => openViewer('original'));
els.zoomResult.addEventListener('click', () => openViewer('result'));
els.downloadOriginal.addEventListener('click', () => downloadImage(state.imagePath));
els.downloadResult.addEventListener('click', () => downloadImage(state.resultPath));
for (const frame of document.querySelectorAll('.frame')) {
  frame.addEventListener('dblclick', () => openViewer(frame.dataset.image));
}
els.clear.addEventListener('click', clearAll);

// Accept drops anywhere in the window; highlight the drop zone while dragging.
document.addEventListener('dragover', (e) => {
  e.preventDefault();
  els.dropZone.classList.add('dragging');
});
document.addEventListener('dragleave', (e) => {
  if (!e.relatedTarget) els.dropZone.classList.remove('dragging');
});
document.addEventListener('drop', (e) => {
  e.preventDefault();
  els.dropZone.classList.remove('dragging');
  const file = e.dataTransfer.files[0];
  if (file) loadImage(window.sr.pathForFile(file));
});

window.sr.onProgress(({ fraction }) => showProgress(fraction, 'Upscaling...'));
window.sr.onModelDownload(onModelDownload);
els.manageModels.addEventListener('click', () => {
  renderModelList();
  els.modelsDialog.showModal();
});
$('close-models').addEventListener('click', () => els.modelsDialog.close());
$('open-models-folder').addEventListener('click', () => window.sr.openModelsFolder());
els.urlForm.addEventListener('submit', addModelFromUrl);
loadGpuPreference();
els.useGpu.addEventListener('change', () => {
  saveGpuPreference();
  updateGpuUI();
});
detectGpus();
refreshModels();

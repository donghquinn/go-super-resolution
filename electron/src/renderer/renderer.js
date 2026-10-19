const IMAGE_EXT = /\.(jpe?g|png|bmp|tiff?)$/i;

const $ = (id) => document.getElementById(id);
const els = {
  dropZone: $('drop-zone'),
  engine: $('engine'),
  mode: $('mode'),
  scale: $('scale'),
  customSize: $('custom-size'),
  width: $('width'),
  height: $('height'),
  model: $('model'),
  modelSection: $('model-section'),
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
  progress: $('progress'),
};

const state = { imagePath: null, resultPath: null, processing: false, models: [], activeModelId: null };

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
  els.save.disabled = true;
  const { naturalWidth: w, naturalHeight: h } = els.originalImg;
  els.originalSize.textContent = `(${w}×${h})`;
  setStatus(`Loaded: ${basename(filePath)} (${w}x${h})`);
}

// (Re)builds the model menu: downloaded models first, then ones that download on first use.
async function refreshModels() {
  const firstLoad = state.models.length === 0;
  const selected = els.modelSelect.value;
  const { models, dir } = await window.sr.listModels();
  state.models = models;
  els.modelsDir.textContent = dir;

  for (const node of [...els.modelSelect.children]) if (node.value !== 'custom') node.remove();
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
  els.modelSelect.value = !firstLoad && (selected === 'custom' || exists(selected)) ? selected : models[0].id;
  updateModelUI();
  if (els.modelsDialog.open) renderModelList();
}

// Managed models are ONNX, so other engines need a custom model file; opencv needs no model.
function updateModelUI() {
  const onnx = els.engine.value === 'onnx';
  els.modelSection.hidden = els.engine.value === 'opencv';
  for (const opt of els.modelSelect.querySelectorAll('option')) opt.disabled = !onnx && opt.value !== 'custom';
  if (!onnx) els.modelSelect.value = 'custom';

  const model = state.models.find((m) => m.id === els.modelSelect.value);
  els.customModel.hidden = Boolean(model);
  els.modelHint.textContent = model ? modelSummary(model) : '';
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
  els.engine.value = 'onnx';
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

  const modelId = els.engine.value === 'onnx' && els.modelSelect.value !== 'custom' ? els.modelSelect.value : null;
  state.activeModelId = modelId;
  try {
    const result = await window.sr.process({
      inputPath: state.imagePath,
      modelId,
      modelPath: els.model.value,
      engine: els.engine.value,
      mode: els.mode.value,
      scale: els.scale.value === 'custom' ? 'custom' : Number(els.scale.value),
      width: Number(els.width.value) || 0,
      height: Number(els.height.value) || 0,
    });
    state.resultPath = result.outputPath;
    await showImage(els.resultImg, result.outputPath);
    els.resultSize.textContent = `(${els.resultImg.naturalWidth}×${els.resultImg.naturalHeight})`;
    els.save.disabled = false;
    const notes = result.notes.length ? ` (${result.notes.join('; ')})` : '';
    setStatus(`Processing completed successfully! Saved to ${basename(result.outputPath)}${notes}`);
  } catch (err) {
    setStatus(`Processing failed: ${errorText(err)}`);
  } finally {
    state.processing = false;
    els.process.disabled = false;
    els.progress.hidden = true;
  }
}

async function saveResult() {
  if (!state.resultPath) return;
  try {
    const saved = await window.sr.saveResult(state.resultPath);
    if (saved) setStatus(`Saved: ${saved}`);
  } catch (err) {
    setStatus(`Error: ${err.message}`);
  }
}

function clearAll() {
  state.imagePath = null;
  state.resultPath = null;
  resetImage(els.originalImg, els.originalSize);
  resetImage(els.resultImg, els.resultSize);
  els.save.disabled = true;
  setStatus('Ready - Select an image to get started');
}

$('browse-image').addEventListener('click', async () => {
  const filePath = await window.sr.openImage();
  if (filePath) loadImage(filePath);
});

$('browse-model').addEventListener('click', async () => {
  const filePath = await window.sr.openModel();
  if (filePath) els.model.value = filePath;
});

els.scale.addEventListener('change', () => {
  els.customSize.hidden = els.scale.value !== 'custom';
});

els.engine.addEventListener('change', updateModelUI);
els.modelSelect.addEventListener('change', updateModelUI);
els.process.addEventListener('click', processImage);
els.save.addEventListener('click', saveResult);
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
if (window.sr.platform === 'darwin') els.mode.value = 'mps';
refreshModels();

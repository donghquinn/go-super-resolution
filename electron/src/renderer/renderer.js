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

const state = { imagePath: null, resultPath: null, processing: false, models: [] };

const basename = (p) => p.split(/[\\/]/).pop();
const setStatus = (text) => { els.status.textContent = text; };

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

// (Re)builds the built-in model options, keeping the current choice (first model on startup).
async function refreshModels() {
  const firstLoad = state.models.length === 0;
  const selected = els.modelSelect.value;
  state.models = await window.sr.listModels();
  for (const opt of [...els.modelSelect.options]) if (opt.value !== 'custom') opt.remove();
  els.modelSelect.prepend(...state.models.map((m) => new Option(m.label, m.id)));
  els.modelSelect.value = firstLoad ? state.models[0].id : selected;
  updateModelUI();
}

// Built-in models are ONNX, so other engines need a custom model file; opencv needs no model.
function updateModelUI() {
  const onnx = els.engine.value === 'onnx';
  els.modelSection.hidden = els.engine.value === 'opencv';
  for (const opt of els.modelSelect.options) opt.disabled = !onnx && opt.value !== 'custom';
  if (!onnx) els.modelSelect.value = 'custom';

  const model = state.models.find((m) => m.id === els.modelSelect.value);
  els.customModel.hidden = Boolean(model);
  els.modelHint.textContent = model
    ? `${model.downloaded ? 'Downloaded' : `Downloads ${model.sizeMB} MB on first use`} · CC-BY-NC-SA-4.0 (non-commercial)`
    : '';
}

function showProgress(info) {
  els.progress.value = info.fraction;
  const percent = Math.round(info.fraction * 100);
  setStatus(info.phase === 'download'
    ? `Downloading ${info.label} (${info.sizeMB} MB)... ${percent}%`
    : `Upscaling... ${percent}%`);
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
    setStatus(`Processing failed: ${err.message.replace(/^Error invoking remote method '[^']+': (Error: )?/, '')}`);
  } finally {
    state.processing = false;
    els.process.disabled = false;
    els.progress.hidden = true;
    if (modelId) refreshModels();
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

window.sr.onProgress(showProgress);
if (window.sr.platform === 'darwin') els.mode.value = 'mps';
refreshModels();

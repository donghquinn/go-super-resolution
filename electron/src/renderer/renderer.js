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

const state = { imagePath: null, resultPath: null, processing: false };

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

async function processImage() {
  if (!state.imagePath) {
    setStatus('Please select an image first');
    return;
  }
  if (state.processing) return;

  state.processing = true;
  els.process.disabled = true;
  els.progress.hidden = false;
  setStatus('Processing...');

  try {
    const result = await window.sr.process({
      inputPath: state.imagePath,
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
    setStatus(`Processing completed successfully! Saved to ${basename(result.outputPath)}`);
  } catch (err) {
    setStatus(`Processing failed: ${err.message.replace(/^Error invoking remote method '[^']+': (Error: )?/, '')}`);
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

// Model management: a catalog of verified ONNX models plus models users add from a URL or by
// dropping .onnx files into the models folder.
const { once } = require('node:events');
const fsSync = require('node:fs');
const fs = require('node:fs/promises');
const path = require('node:path');

// Verified with sr-backend (tiling, precision and output checked against ONNX Runtime in Python).
const CATALOG = [
  {
    id: 'ultrasharp-v2',
    label: 'UltraSharpV2',
    description: 'Best overall quality for photos, art and anime',
    arch: 'DAT2',
    scale: 4,
    sizeMB: 52,
    license: 'CC-BY-NC-SA-4.0',
    author: 'Kim2091',
    file: '4x-UltraSharpV2_fp32_op17.onnx',
    url: 'https://huggingface.co/Kim2091/UltraSharpV2/resolve/main/4x-UltraSharpV2_fp32_op17.onnx',
  },
  {
    id: 'ultrasharp-v2-lite',
    label: 'UltraSharpV2 Lite',
    description: 'Fast version of UltraSharpV2',
    arch: 'RealPLKSR',
    scale: 4,
    sizeMB: 30,
    license: 'CC-BY-NC-SA-4.0',
    author: 'Kim2091',
    file: '4x-UltraSharpV2_Lite_fp32_op17.onnx',
    url: 'https://huggingface.co/Kim2091/UltraSharpV2/resolve/main/4x-UltraSharpV2_Lite_fp32_op17.onnx',
  },
  {
    id: 'realwebphoto-v4-dat2',
    label: 'RealWebPhoto v4',
    description: 'Photos from the web: JPEG artifacts, noise, blur',
    arch: 'DAT2',
    scale: 4,
    sizeMB: 49,
    license: 'CC-BY-4.0',
    author: 'Phhofm',
    file: '4xRealWebPhoto_v4_dat2_fp32_opset17.onnx',
    url: 'https://github.com/Phhofm/models/releases/download/4xRealWebPhoto_v4_dat2/4xRealWebPhoto_v4_dat2_fp32_opset17.onnx',
  },
  {
    id: 'nomos2-realplksr',
    label: 'Nomos2 RealPLKSR',
    description: 'Fast photo upscaler, commercial use allowed',
    arch: 'RealPLKSR',
    scale: 4,
    sizeMB: 30,
    license: 'CC-BY-4.0',
    author: 'Phhofm',
    file: '4xNomos2_realplksr_dysample_256_fp32_fullyoptimized.onnx',
    url: 'https://github.com/Phhofm/models/releases/download/4xNomos2_realplksr_dysample/4xNomos2_realplksr_dysample_256_fp32_fullyoptimized.onnx',
  },
  {
    id: 'realplksr-x2',
    label: 'RealPLKSR 2x',
    description: 'Native 2x model, faithful (non-GAN) output',
    arch: 'RealPLKSR',
    scale: 2,
    sizeMB: 30,
    license: 'MIT',
    author: 'darktable',
    file: 'realplksr_x2.onnx',
    url: 'https://huggingface.co/darktable-org/upscale-realplksr-onnx/resolve/main/onnx/model_x2.onnx',
  },
  {
    id: 'realplksr-x4',
    label: 'RealPLKSR 4x',
    description: 'Faithful (non-GAN) 4x output, commercial use allowed',
    arch: 'RealPLKSR',
    scale: 4,
    sizeMB: 30,
    license: 'MIT',
    author: 'darktable',
    file: 'realplksr_x4.onnx',
    url: 'https://huggingface.co/darktable-org/upscale-realplksr-onnx/resolve/main/onnx/model_x4.onnx',
  },
];

const INDEX_FILE = 'models.json'; // details of models added from a URL, keyed by file name
const USER_PREFIX = 'file:';

// Turns a pasted link into a direct download URL and a safe file name.
function parseModelUrl(raw) {
  let url;
  try {
    url = new URL(String(raw).trim());
  } catch {
    throw new Error('Enter a valid URL');
  }
  if (url.protocol !== 'https:' && url.protocol !== 'http:') throw new Error('Only http(s) URLs are supported');
  // Page links on Hugging Face and GitHub point at HTML; use the raw file instead.
  if (url.hostname === 'huggingface.co') url.pathname = url.pathname.replace('/blob/', '/resolve/');
  if (url.hostname === 'github.com') url.pathname = url.pathname.replace('/blob/', '/raw/');

  const name = decodeURIComponent(path.posix.basename(url.pathname));
  if (!/\.onnx$/i.test(name)) throw new Error('The URL must point to an .onnx file');
  const file = name.replace(/[^\w.-]+/g, '_');
  if (file === INDEX_FILE || file.startsWith('.')) throw new Error('Invalid model file name');
  return { url: url.href, file };
}

class ModelStore {
  // inspect(path) checks a model with the backend; onEvent receives download progress/completion.
  constructor(dir, { inspect, onEvent }) {
    this.dir = dir;
    this.inspect = inspect;
    this.onEvent = onEvent;
    this.downloads = new Map(); // id -> { promise, controller, label, fraction }
  }

  pathOf(model) {
    return path.join(this.dir, model.file);
  }

  async files() {
    try {
      return (await fs.readdir(this.dir)).filter((f) => /\.onnx$/i.test(f));
    } catch (err) {
      if (err.code === 'ENOENT') return [];
      throw err;
    }
  }

  async readIndex() {
    try {
      return JSON.parse(await fs.readFile(path.join(this.dir, INDEX_FILE), 'utf8'));
    } catch {
      return {};
    }
  }

  async updateIndex(fn) {
    const index = await this.readIndex();
    fn(index);
    await fs.writeFile(path.join(this.dir, INDEX_FILE), `${JSON.stringify(index, null, 2)}\n`);
  }

  // Catalog models (downloaded or not), then user models, then URL downloads in progress.
  async list() {
    const files = await this.files();
    const index = await this.readIndex();
    const progress = (id) => this.downloads.get(id)?.fraction;
    const catalog = CATALOG.map(({ url, file, ...m }) => ({
      ...m,
      source: 'catalog',
      downloaded: files.includes(file),
      downloading: this.downloads.has(m.id),
      progress: progress(m.id),
    }));
    const user = await Promise.all(files
      .filter((f) => !CATALOG.some((m) => m.file === f))
      .map(async (file) => {
        const id = USER_PREFIX + file;
        const { size } = await fs.stat(path.join(this.dir, file));
        return {
          id,
          label: file.replace(/\.onnx$/i, ''),
          source: 'user',
          sizeMB: Math.round(size / 1e6),
          downloaded: !this.downloads.has(id),
          downloading: this.downloads.has(id),
          progress: progress(id),
          ...index[file],
        };
      }));
    const pending = [...this.downloads]
      .filter(([id]) => id.startsWith(USER_PREFIX) && !files.includes(id.slice(USER_PREFIX.length)))
      .map(([id, d]) => ({ id, label: d.label, source: 'user', downloaded: false, downloading: true, progress: d.fraction }));
    return [...catalog, ...user, ...pending];
  }

  // Resolves a model id from the renderer; user ids only match files that exist in the folder.
  async find(id) {
    const entry = CATALOG.find((m) => m.id === id);
    if (entry) return entry;
    const file = String(id).startsWith(USER_PREFIX) ? id.slice(USER_PREFIX.length) : null;
    if (file && (await this.files()).includes(file)) {
      return { id, file, label: file.replace(/\.onnx$/i, '') };
    }
    return null;
  }

  // Returns the model's path, downloading a catalog model first if needed.
  async ensure(id) {
    const model = await this.find(id);
    if (!model) throw new Error(`Unknown model: ${id}`);
    if (!fsSync.existsSync(this.pathOf(model))) await this.download(model.id, model.url, model.file, model.label);
    return this.pathOf(model);
  }

  // Starts (or joins) a download; concurrent requests for the same model share one transfer.
  download(id, url, file, label) {
    if (this.downloads.has(id)) return this.downloads.get(id).promise;
    const controller = new AbortController();
    const entry = { controller, label, fraction: 0 };
    entry.promise = this.transfer(url, path.join(this.dir, file), controller.signal, (fraction) => {
      entry.fraction = fraction;
      this.onEvent({ id, label, fraction });
    }).then(
      () => { this.downloads.delete(id); this.onEvent({ id, done: true }); },
      (err) => {
        this.downloads.delete(id);
        const error = controller.signal.aborted ? new Error('Download canceled') : err;
        this.onEvent({ id, done: true, error: error.message });
        throw error;
      },
    );
    this.downloads.set(id, entry);
    return entry.promise;
  }

  async transfer(url, dest, signal, onProgress) {
    const part = `${dest}.part`;
    await fs.mkdir(path.dirname(dest), { recursive: true });
    try {
      const res = await fetch(url, { signal });
      if (!res.ok) throw new Error(`Download failed: HTTP ${res.status}`);
      const total = Number(res.headers.get('content-length')) || 0;
      const out = fsSync.createWriteStream(part);
      try {
        let received = 0;
        let reported = -1;
        for await (const chunk of res.body) {
          received += chunk.length;
          if (!out.write(chunk)) await once(out, 'drain');
          const percent = total ? Math.floor((received / total) * 100) : 0;
          if (percent !== reported) onProgress(total ? percent / 100 : null);
          reported = percent;
        }
        if (total && received !== total) throw new Error('Download was incomplete');
      } finally {
        await new Promise((resolve) => out.end(resolve));
      }
      await fs.rename(part, dest);
    } catch (err) {
      await fs.rm(part, { force: true });
      throw err;
    }
  }

  cancel(id) {
    this.downloads.get(id)?.controller.abort();
  }

  async remove(id) {
    this.cancel(id);
    const model = await this.find(id);
    if (!model) return;
    await fs.rm(this.pathOf(model), { force: true });
    if (model.id.startsWith(USER_PREFIX)) await this.updateIndex((index) => { delete index[model.file]; });
  }

  // Downloads an ONNX model from a URL, keeping it only if the backend can run it.
  async addFromUrl(raw) {
    const { url, file } = parseModelUrl(raw);
    const known = CATALOG.find((m) => m.url === url || m.file === file);
    if (known) {
      await this.ensure(known.id);
      return { id: known.id };
    }
    const id = USER_PREFIX + file;
    const dest = path.join(this.dir, file);
    if (fsSync.existsSync(dest)) throw new Error(`${file} is already in the models folder`);

    await this.download(id, url, file, file.replace(/\.onnx$/i, ''));
    let info;
    try {
      info = await this.inspect(dest);
    } catch (err) {
      await fs.rm(dest, { force: true });
      this.onEvent({ id, done: true });
      throw new Error(`Not a usable super-resolution model: ${err.message}`);
    }
    await this.updateIndex((index) => {
      index[file] = { url, scale: info.scale, precision: info.precision, addedAt: new Date().toISOString() };
    });
    this.onEvent({ id, done: true });
    return { id, ...info };
  }
}

module.exports = { CATALOG, ModelStore, parseModelUrl };

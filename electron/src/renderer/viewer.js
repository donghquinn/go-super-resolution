// Full-window zoomable image viewer: the original, the result, or both side by side with zoom
// and pan kept in sync.
//
// Both images are laid out in one shared coordinate space, the result's pixel grid (or the
// original's when there is no result). The original is stretched onto it, so the same point
// lines up in both panes. state.scale maps that space to CSS pixels; x/y offset it in the pane.
class ImageViewer {
  static MAX_SCALE = 64;

  constructor(dialog, { onDownload }) {
    this.dialog = dialog;
    this.onDownload = onDownload;
    this.panes = {
      original: dialog.querySelector('[data-pane="original"]'),
      result: dialog.querySelector('[data-pane="result"]'),
    };
    this.zoomLabel = dialog.querySelector('#viewer-zoom');
    this.viewButtons = [...dialog.querySelectorAll('[data-view]')];
    this.state = { scale: 1, x: 0, y: 0 };
    this.fitted = true;
    this.sources = {};
    this.view = 'result';
    this.bindEvents();
  }

  // sources: { original: {path, url, width, height}, result: {...} | null }
  open(sources, view) {
    this.sources = sources;
    for (const [name, pane] of Object.entries(this.panes)) {
      const img = pane.querySelector('img');
      if (sources[name]) img.src = sources[name].url;
      else img.removeAttribute('src');
    }
    for (const b of this.viewButtons) b.disabled = b.dataset.view !== 'original' && !sources.result;
    this.dialog.showModal();
    this.setView(sources[view] ? view : 'original', { refit: true });
  }

  get base() {
    const s = this.sources.result || this.sources.original;
    return { w: s.width, h: s.height };
  }

  visiblePanes() {
    return this.view === 'compare' ? [this.panes.original, this.panes.result] : [this.panes[this.view]];
  }

  // Size of one pane; in side-by-side view both panes are the same size.
  viewport() {
    const pane = this.visiblePanes()[0];
    return { w: pane.clientWidth, h: pane.clientHeight };
  }

  setView(view, { refit = false } = {}) {
    const center = refit ? null : this.viewportCenterInBase();
    this.view = view;
    for (const [name, pane] of Object.entries(this.panes)) {
      pane.hidden = !(view === 'compare' || view === name);
    }
    for (const b of this.viewButtons) b.setAttribute('aria-pressed', String(b.dataset.view === view));
    if (refit || this.fitted) this.fit();
    else this.centerOn(center);
  }

  viewportCenterInBase() {
    const vp = this.viewport();
    const { scale, x, y } = this.state;
    return { x: (vp.w / 2 - x) / scale, y: (vp.h / 2 - y) / scale };
  }

  centerOn(point) {
    const vp = this.viewport();
    this.state.x = vp.w / 2 - point.x * this.state.scale;
    this.state.y = vp.h / 2 - point.y * this.state.scale;
    this.apply();
  }

  fit() {
    const vp = this.viewport();
    const { w, h } = this.base;
    const scale = Math.min(vp.w / w, vp.h / h);
    this.state = { scale, x: (vp.w - w * scale) / 2, y: (vp.h - h * scale) / 2 };
    this.fitted = true;
    this.apply();
  }

  // 100%: one pixel of the shown image (the result in side-by-side view) per CSS pixel.
  actualScale() {
    const shown = this.sources[this.view === 'original' ? 'original' : 'result'];
    return shown.width / this.base.w;
  }

  zoomAt(factor, cx, cy) {
    const vp = this.viewport();
    const fitScale = Math.min(vp.w / this.base.w, vp.h / this.base.h);
    const { scale, x, y } = this.state;
    const next = Math.min(Math.max(scale * factor, Math.min(fitScale, this.actualScale()) / 4), ImageViewer.MAX_SCALE);
    this.state = { scale: next, x: cx - ((cx - x) * next) / scale, y: cy - ((cy - y) * next) / scale };
    this.fitted = false;
    this.apply();
  }

  zoomCentered(factor) {
    const vp = this.viewport();
    this.zoomAt(factor, vp.w / 2, vp.h / 2);
  }

  apply() {
    const { scale, x, y } = this.state;
    const { w, h } = this.base;
    for (const [name, pane] of Object.entries(this.panes)) {
      const img = pane.querySelector('img');
      const source = this.sources[name];
      if (!source) continue;
      img.style.width = `${w}px`;
      img.style.height = `${h}px`;
      img.style.transform = `translate(${x}px, ${y}px) scale(${scale})`;
      // Show real pixels (no smoothing) once they're magnified enough to see.
      img.classList.toggle('pixelated', (scale * w) / source.width >= 2);
    }
    this.zoomLabel.value = `${Math.round((scale / this.actualScale()) * 100)}%`;
  }

  bindEvents() {
    const d = this.dialog;
    d.querySelector('#viewer-close').addEventListener('click', () => d.close());
    d.querySelector('#viewer-fit').addEventListener('click', () => this.fit());
    d.querySelector('#viewer-actual').addEventListener('click', () => this.zoomCentered(this.actualScale() / this.state.scale));
    d.querySelector('#viewer-zoom-in').addEventListener('click', () => this.zoomCentered(1.5));
    d.querySelector('#viewer-zoom-out').addEventListener('click', () => this.zoomCentered(1 / 1.5));
    d.querySelector('#viewer-download').addEventListener('click', () => {
      const name = this.view === 'original' ? 'original' : 'result';
      this.onDownload(this.sources[name].path);
    });
    for (const b of this.viewButtons) b.addEventListener('click', () => this.setView(b.dataset.view));

    for (const pane of Object.values(this.panes)) {
      const local = (e) => {
        const r = pane.getBoundingClientRect();
        return [e.clientX - r.left, e.clientY - r.top];
      };
      // Mouse wheel and trackpad pinch (reported as ctrl+wheel) zoom around the cursor.
      pane.addEventListener('wheel', (e) => {
        e.preventDefault();
        this.zoomAt(Math.exp(-e.deltaY * (e.ctrlKey ? 0.01 : 0.002)), ...local(e));
      }, { passive: false });

      pane.addEventListener('pointerdown', (e) => {
        if (e.button !== 0) return;
        pane.setPointerCapture(e.pointerId);
        let last = [e.clientX, e.clientY];
        pane.classList.add('dragging');
        const move = (ev) => {
          this.state.x += ev.clientX - last[0];
          this.state.y += ev.clientY - last[1];
          last = [ev.clientX, ev.clientY];
          this.fitted = false;
          this.apply();
        };
        const up = () => {
          pane.classList.remove('dragging');
          pane.removeEventListener('pointermove', move);
          pane.removeEventListener('pointerup', up);
          pane.removeEventListener('pointercancel', up);
        };
        pane.addEventListener('pointermove', move);
        pane.addEventListener('pointerup', up);
        pane.addEventListener('pointercancel', up);
      });

      // Double-click: zoom to 100% at the cursor, or back to fit if already zoomed in.
      pane.addEventListener('dblclick', (e) => {
        const actual = this.actualScale();
        if (this.state.scale < actual * 0.99) this.zoomAt(actual / this.state.scale, ...local(e));
        else this.fit();
      });
    }

    d.addEventListener('keydown', (e) => {
      if (e.target.closest('button') && (e.key === 'Enter' || e.key === ' ')) return;
      const actions = {
        '+': () => this.zoomCentered(1.5),
        '=': () => this.zoomCentered(1.5),
        '-': () => this.zoomCentered(1 / 1.5),
        0: () => this.fit(),
        1: () => this.zoomCentered(this.actualScale() / this.state.scale),
        ArrowLeft: () => this.pan(80, 0),
        ArrowRight: () => this.pan(-80, 0),
        ArrowUp: () => this.pan(0, 80),
        ArrowDown: () => this.pan(0, -80),
      };
      if (actions[e.key]) {
        e.preventDefault();
        actions[e.key]();
      }
    });

    window.addEventListener('resize', () => {
      if (!d.open) return;
      if (this.fitted) this.fit();
      else this.apply();
    });
  }

  pan(dx, dy) {
    this.state.x += dx;
    this.state.y += dy;
    this.fitted = false;
    this.apply();
  }
}

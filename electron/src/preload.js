const { contextBridge, ipcRenderer, webUtils } = require('electron');

contextBridge.exposeInMainWorld('sr', {
  openImage: () => ipcRenderer.invoke('open-image'),
  openModel: () => ipcRenderer.invoke('open-model'),
  saveResult: (resultPath) => ipcRenderer.invoke('save-result', resultPath),
  listModels: () => ipcRenderer.invoke('list-models'),
  listDevices: () => ipcRenderer.invoke('list-devices'),
  downloadModel: (id) => ipcRenderer.invoke('download-model', id),
  cancelDownload: (id) => ipcRenderer.invoke('cancel-download', id),
  deleteModel: (id) => ipcRenderer.invoke('delete-model', id),
  addModelUrl: (url) => ipcRenderer.invoke('add-model-url', url),
  openModelsFolder: () => ipcRenderer.invoke('open-models-folder'),
  onModelDownload: (callback) => ipcRenderer.on('model-download', (_e, event) => callback(event)),
  process: (opts) => ipcRenderer.invoke('process', opts),
  onProgress: (callback) => ipcRenderer.on('progress', (_e, info) => callback(info)),
  fileUrl: (filePath) => ipcRenderer.invoke('file-url', filePath),
  // File.path was removed from Electron; dropped files need webUtils to get a real path.
  pathForFile: (file) => webUtils.getPathForFile(file),
});

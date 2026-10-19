const { contextBridge, ipcRenderer, webUtils } = require('electron');

contextBridge.exposeInMainWorld('sr', {
  openImage: () => ipcRenderer.invoke('open-image'),
  openModel: () => ipcRenderer.invoke('open-model'),
  saveResult: (resultPath) => ipcRenderer.invoke('save-result', resultPath),
  process: (opts) => ipcRenderer.invoke('process', opts),
  fileUrl: (filePath) => ipcRenderer.invoke('file-url', filePath),
  // File.path was removed from Electron; dropped files need webUtils to get a real path.
  pathForFile: (file) => webUtils.getPathForFile(file),
});

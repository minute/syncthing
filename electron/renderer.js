const {ipcRenderer} = require('electron')

ipcRenderer.on('startup', (event, arg) => {
  console.log(arg)
})

const electron = require('electron')
const child_process = require('child_process')
const app = electron.app
const BrowserWindow = electron.BrowserWindow
const url = require('url')
const path = require('path')

let win
let st, stURL, stAPIKey

app.on('ready', createWindow)

function createWindow() {
  win = new BrowserWindow({ width: 1200, height: 800 })

  win.loadURL(url.format({
    pathname: path.join(__dirname, 'index.html'),
    protocol: 'file:',
    slashes: true
  }))

  if (st) {
    win.webContents.loadURL(stURL, { extraHeaders: `X-API-Key: ${stAPIKey}` })
  } else {
    startSyncthing()
  }

  win.on('closed', () => {
    win = null
  })
}

app.on('activate', () => {
  if (win === null) {
    createWindow()
  }
})

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

function startSyncthing() {
  stAPIKey = stAPIKey || randomAPIKey()
  st = child_process.spawn(`${__dirname}/bin/syncthing`, ['-no-browser'], {
    env: {
      "STGUIADDRESS": stURL || "http://127.0.0.1:0",
      "STGUIAPIKEY": stAPIKey,
      "STNORESTART": "1",
      "HOME": process.env["HOME"],
    }
  })

  st.on('exit', (code, signal) => {
    console.log(code, signal)
    if (code === 0) {
      // Shutdown
      app.quit()
    } else {
      // Restarting
      startSyncthing()
    }
  })

  st.stdout.on('data', (data) => {
    process.stderr.write(data)
    const m = ('' + data).match("API listening on ([0-9:.]+)")
    if (m) {
      stURL = `http://${m[1]}/`
      win.webContents.loadURL(stURL, { extraHeaders: `X-API-Key: ${stAPIKey}` })
    }
  })
}

function randomAPIKey() {
  var text = '';
  var possible = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';

  for (var i = 0; i < 32; i++)
    text += possible.charAt(Math.floor(Math.random() * possible.length));

  return text;
}
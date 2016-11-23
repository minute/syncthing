const electron = require('electron')
const child_process = require('child_process')
const app = electron.app
const BrowserWindow = electron.BrowserWindow
const url = require('url')
const path = require('path')

let win
let stURL, stAPIKey

/*
var menubar = require('menubar')

var mb = menubar({
  index: `file://${__dirname}/popup.html`,
  width: 200,
  height: 400,
})

mb.on('ready', function ready () {
  startSyncthing(mb);
})
*/

app.on('ready', createWindow)

function startSyncthing() {
  var apiKey = randomAPIKey()
  const st = child_process.spawn("/Users/jb/bin/syncthing", ["-no-browser"], {env: {
    "STGUIADDRESS": "http://127.0.0.1:0",
    "STGUIAPIKEY": apiKey,
    "STNORESTART": "1",
    "HOME": process.env["HOME"],
  }})
  var s = ''
  st.stdout.on('data', (data) => {
    const m = (''+data).match("API listening on ([0-9:.]+)")
    if (m) {
      stURL = `http://${m[1]}/`
      stAPIKey = apiKey
      win.webContents.loadURL(stURL, {extraHeaders: "X-API-Key: " + stAPIKey})
    }
  })
}

function createWindow () {
  // Create the browser window.
  win = new BrowserWindow({width: 1200, height: 800})

  // and load the index.html of the app.
  win.loadURL(url.format({
    pathname: path.join(__dirname, 'index.html'),
    protocol: 'file:',
    slashes: true
  }))

  // Open the DevTools.
  //win.webContents.openDevTools()

  if (stURL) {
          win.webContents.loadURL(stURL, {extraHeaders: "X-API-Key: " + stAPIKey})
  } else {
    startSyncthing()
  }

  // Emitted when the window is closed.
  win.on('closed', () => {
    // Dereference the window object, usually you would store windows
    // in an array if your app supports multi windows, this is the time
    // when you should delete the corresponding element.
    win = null
  })
}

app.on('activate', () => {
  // On macOS it's common to re-create a window in the app when the
  // dock icon is clicked and there are no other windows open.
  if (win === null) {
    createWindow()
  }
})

app.on('window-all-closed', () => {
  // On macOS it is common for applications and their menu bar
  // to stay active until the user quits explicitly with Cmd + Q
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

function randomAPIKey()
{
    var text = "";
    var possible = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

    for( var i=0; i < 32; i++ )
        text += possible.charAt(Math.floor(Math.random() * possible.length));

    return text;
}
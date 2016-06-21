const electron = require('electron')
const child_process = require('child_process')
const app = electron.app
const BrowserWindow = electron.BrowserWindow

let mainWindow

var menubar = require('menubar')

var mb = menubar({
  index: `file://${__dirname}/popup.html`,
  width: 200,
  height: 400,
})

mb.on('ready', function ready () {
  startSyncthing(mb);
})

function startSyncthing(mb) {
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
      global.mainURL = `http://${m[1]}/`
      global.apiKey = apiKey
    }
  })
}

function randomAPIKey()
{
    var text = "";
    var possible = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

    for( var i=0; i < 32; i++ )
        text += possible.charAt(Math.floor(Math.random() * possible.length));

    return text;
}
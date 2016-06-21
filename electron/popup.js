const remote = require('electron').remote
const {BrowserWindow} = remote;
let mainWindow

function init() {
    update()
}

function update() {
    var oReq = new XMLHttpRequest();
    oReq.addEventListener("load", updateStatus);
    oReq.open("GET", remote.getGlobal("mainURL") + "/rest/system/status")
    oReq.setRequestHeader("X-API-Key", remote.getGlobal("apiKey"))
    oReq.send()

    setTimeout(update, 10000)
}

function updateStatus() {
    var data = JSON.parse(this.responseText)
    document.getElementById("cpu").innerHTML = ''+(100*data.cpuPercent)
    document.getElementById("ram").innerHTML = ''+(data.alloc/1e6)
}

function createWindow () {
    mainWindow = new BrowserWindow({width: 1200, height: 800, webPreferences: {nodeIntegration: false}})
    mainWindow.loadURL(remote.getGlobal("mainURL"))
    mainWindow.on('closed', function () {
        mainWindow = null
    })
}
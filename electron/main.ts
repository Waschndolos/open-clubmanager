import { app, BrowserWindow } from 'electron'
import * as path from 'path'
import { spawn } from 'child_process'
import * as url from 'url'

let mainWindow: BrowserWindow | null = null
let backendProcess: ReturnType<typeof spawn> | null = null

const isDev = !app.isPackaged

function createWindow() {
    if (mainWindow !== null) return

    mainWindow = new BrowserWindow({
        width: 1200,
        height: 800,
        webPreferences: {
            contextIsolation: true
        }
    })

    if (isDev) {
        mainWindow.loadURL('http://localhost:5173') // vite dev server
    } else {
        const indexPath = path.join(__dirname, '../frontend/dist/index.html')
        mainWindow.loadURL(
            url.pathToFileURL(indexPath).toString()
        )
    }

    mainWindow.on('closed', () => {
        mainWindow = null
    })
}

function startBackend() {
    if (backendProcess !== null) return

    const serverEntry = path.join(__dirname, '../backend/dist/server.js')

    backendProcess = spawn(process.execPath, [serverEntry], {
        stdio: 'inherit'
    })

    backendProcess.on('close', (code) => {
        console.log(`Backend process exited with code ${code}`)
    })
}

app.whenReady().then(() => {
    if (!isDev) startBackend()
    createWindow()
})

app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow()
})

app.on('window-all-closed', () => {
    if (process.platform !== 'darwin') {
        if (backendProcess) {
            backendProcess.kill()
        }
        app.quit()
    }
})

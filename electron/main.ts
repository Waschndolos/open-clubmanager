import {app, BrowserWindow} from 'electron'
import * as path from 'path'
import {spawn} from 'child_process'
import * as fs from 'fs'

let mainWindow: BrowserWindow | null = null
let backendProcess: ReturnType<typeof spawn> | null = null
let hasWindowBeenCreated = false
const isDev = !app.isPackaged
console.log('🔁 main.ts executed!')
function createWindow() {
    if (hasWindowBeenCreated) return;
    if (mainWindow !== null) {
        console.log('⚠️  createWindow called but mainWindow exists')
        return
    }
    hasWindowBeenCreated = true

    console.log('🪟 Creating new BrowserWindow...')

    mainWindow = new BrowserWindow({
        width: 1200,
        height: 800,
        webPreferences: {
            contextIsolation: true
        }
    })
    mainWindow.webContents.on('did-fail-load', (_event, errorCode, errorDescription) => {
        console.error(`❌ Failed to load: ${errorCode} - ${errorDescription}`)
    })
    mainWindow.on('unresponsive', () => {
        console.warn('⚠️ Window became unresponsive')
    })


    if (isDev) {
        mainWindow.loadURL('http://localhost:5173')
    } else {
        const indexPath = path.join(__dirname, '..', '..', 'frontend', 'dist', 'index.html')

        if (fs.existsSync(indexPath)) {
            mainWindow.loadFile(indexPath)
        } else {
            console.error('❌ index.html not found at', indexPath)
        }
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

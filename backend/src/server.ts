import express from 'express'
import cors from 'cors'
import { PrismaClient } from '@prisma/client'
import memberRoutes from './routes/member.js'

const app = express()
const prisma = new PrismaClient()

// Middlewares
app.use(cors())
app.use(express.json())

// Routen
app.use('/api/members', memberRoutes)

// Fehlerbehandlung
app.use((err: any, _req: express.Request, res: express.Response, _next: express.NextFunction) => {
    console.error(err)
    res.status(500).json({ error: 'Serverfehler' })
})

// Start
const PORT = process.env.PORT || 3001
app.listen(PORT, () => {
    console.log(`✅ Server läuft auf http://localhost:${PORT}`)
})

import express, {Request, Response} from 'express'
import cors from 'cors'
import memberRoutes from './routes/member.js'

const app = express()

// Middlewares
app.use(cors())
app.use(express.json())

// Routes
app.use('/api/members', memberRoutes)

// Error handling middleware
app.use((err: unknown, _req: Request, res: Response) => {
    console.error(err)
    res.status(500).json({ error: 'Internal server error' })
})

// Start server
const PORT = process.env.PORT || 3001
app.listen(PORT, () => {
    console.log(`✅ Server is running at http://localhost:${PORT}`)
})

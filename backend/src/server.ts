import express, {Request, Response} from 'express'
import cors from 'cors'
import memberRoutes from './routes/member.ts'
import roleRoutes from "./routes/role.ts";
import groupRoutes from "./routes/group.ts";
import sectionRoutes from "./routes/section.ts";
import preferenceRoutes from "./routes/userpreference.ts";
import validationRoutes from "./routes/validation.ts";


const app = express()

console.log("Setup middlewares")
// Middlewares
app.use(cors())
app.use(express.json())

console.log("Setup routes")
// Routes
app.use('/api/members', memberRoutes)
app.use('/api/roles', roleRoutes);
app.use('/api/groups', groupRoutes);
app.use('/api/sections', sectionRoutes);
app.use('/api/preference', preferenceRoutes);
app.use('/api/validation', validationRoutes);

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

import express, {Request, Response} from 'express'
import cors from 'cors'
import memberRoutes from './routes/member'
import roleRoutes from "./routes/role";
import groupRoutes from "./routes/group";
import sectionRoutes from "./routes/section";
import preferenceRoutes from "./routes/userpreference";

process.on('uncaughtException', (err) => {
    console.error('Uncaught Exception:', {
        message: err.message,
        stack: err.stack,
        name: err.name,
    });
});
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
// app.use('/api/settings', settingRoutes);
// app.use('/api/statistics', statisticRoutes);

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

import express, {Request, Response} from 'express'
import cookieParser from "cookie-parser";
import cors from 'cors'
import memberRoutes from './routes/member.ts'
import roleRoutes from "./routes/role.ts";
import groupRoutes from "./routes/group.ts";
import sectionRoutes from "./routes/section.ts";
import preferenceRoutes from "./routes/userpreference.ts";
import validationRoutes from "./routes/validation.ts";
import settingRoutes from "./routes/settings.ts";
import statisticRoutes from "./routes/statistics.ts";
import authRoutes from "./routes/auth.ts";

process.on('uncaughtException', (err) => {
    console.error('Uncaught Exception:', {
        message: err.message,
        stack: err.stack,
        name: err.name,
    });
});
const allowedOrigins = ["http://localhost:5173"];
const app = express()

console.log("Setup middlewares")
// Middlewares
app.use(cors({
    origin: function(origin, callback){
        if (!origin) return callback(null, true);
        if (allowedOrigins.indexOf(origin) === -1) {
            const msg = 'The CORS policy for this site does not allow access from the specified origin.';
            return callback(new Error(msg), false);
        }
        return callback(null, true);
    },
    credentials: true,
}));
app.use(express.json())
app.use(cookieParser())

console.log("Setup routes")
// Routes
app.use('/api/members', memberRoutes)
app.use('/api/roles', roleRoutes);
app.use('/api/groups', groupRoutes);
app.use('/api/sections', sectionRoutes);
app.use('/api/preference', preferenceRoutes);
app.use('/api/validation', validationRoutes);
app.use('/api/settings', settingRoutes);
app.use('/api/statistics', statisticRoutes);
app.use('/api/auth', authRoutes);

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

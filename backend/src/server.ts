import express, {Request, Response} from 'express'
import cookieParser from "cookie-parser";
import cors from 'cors'
import { createApiV2Router } from './v2/createApiV2Router.ts';

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
app.use('/api/v2', createApiV2Router());

// Error handling middleware
app.use((err: unknown, _req: Request, res: Response, _next: express.NextFunction) => {
    console.error(err);
    res.status(500).json({ error: 'Internal server error' });
});

// Start server
const PORT = process.env.PORT || 3001
app.listen(PORT, () => {
    console.log(`✅ Server is running at http://localhost:${PORT}`)
})

import { SetupRepository } from './setupRepository.ts';
import { HttpError } from '../../core/httpError.ts';
import bcrypt from 'bcrypt';

export interface SetupStatus {
    setupRequired: boolean;
    userCount: number;
}

export class SetupService {
    constructor(private readonly setupRepository: SetupRepository) {}

    async getStatus(): Promise<SetupStatus> {
        const userCount = await this.setupRepository.countUsers();

        return {
            setupRequired: userCount === 0,
            userCount,
        };
    }

    async initializeAdmin(email: string, password: string): Promise<void> {
        const normalizedEmail = email.trim().toLowerCase();

        if (!normalizedEmail || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizedEmail)) {
            throw new HttpError(400, 'SETUP_INVALID_EMAIL', 'A valid email address is required.');
        }

        if (!password || password.length < 8) {
            throw new HttpError(400, 'SETUP_INVALID_PASSWORD', 'Password must be at least 8 characters long.');
        }

        const userCount = await this.setupRepository.countUsers();

        if (userCount > 0) {
            throw new HttpError(403, 'SETUP_ALREADY_COMPLETED', 'Setup has already been completed.');
        }

        const passwordHash = await bcrypt.hash(password, 10);
        await this.setupRepository.createInitialAdmin(normalizedEmail, passwordHash);
    }
}


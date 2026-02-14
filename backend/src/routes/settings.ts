import {FileStorage} from '../services/filestorage';

export interface Settings {
    defaultCurrency: 'usd' | 'eur' | 'gbp' | 'chf';
    showWelcomeBanner: boolean;
    itemsPerPage: number;
    [extra: string]: unknown;
}

class SettingsService {
    private readonly storage = new FileStorage<Settings>('data/settings');
    private cache: Settings | null = null;

    /** Load settings from disk (cached after first read) */
    async get(): Promise<Settings> {
        if (this.cache) return this.cache;

        const persisted = await this.storage.read('global');
        // If the file does not exist yet we fall back to sensible defaults
        this.cache = persisted ?? {
            defaultCurrency: 'eur',
            showWelcomeBanner: true,
            itemsPerPage: 25,
        };
        return this.cache;
    }

    async set(newSettings: Settings): Promise<void> {
        this.cache = newSettings;
        await this.storage.write('global', newSettings);
    }

    async update<K extends keyof Settings>(key: K, value: Settings[K]): Promise<void> {
        const current = await this.get();
        current[key] = value;
        await this.set(current);
    }
}

export const settings = new SettingsService();
import {UserPreference} from "../models/models.ts";
import {Repository} from "./repository.ts";
import {FileStorage} from "../services/filestorage.ts";

export class UserPreferenceRepository
    implements Repository<UserPreference, string>
{
    private readonly storage = new FileStorage<UserPreference>('data/userPreferences');

    async findById(id: string) {
        return this.storage.read(id);
    }

    async findAll(filter?: Partial<UserPreference>) {
        const ids = await this.storage.list();
        const prefs = await Promise.all(ids.map(id => this.storage.read(id)));
        const cleaned = prefs.filter((p): p is UserPreference => p !== null);

        if (!filter) return cleaned;
        return cleaned.filter(p =>
            Object.entries(filter).every(([k, v]) => (p as any)[k] === v)
        );
    }

    async upsert(pref: UserPreference) {
        const all = await this.findAll({ userId: pref.userId, key: pref.key });
        if (all.length > 0) {
            pref.id = all[0].id;
        } else {
            const existing = await this.storage.list();
            const maxId = existing
                .map(id => Number(id))
                .reduce((a, b) => (a > b ? a : b), 0);
            pref.id = maxId + 1;
        }

        await this.storage.write(String(pref.id), pref);
        return pref;
    }

    async delete(id: string) {
        await this.storage.delete(id);
    }
}
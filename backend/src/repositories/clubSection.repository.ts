import {ClubSection} from "../models/models.ts";
import {FileStorage} from "../services/filestorage.ts";
import {Repository} from "./repository.ts";

export class ClubSectionRepository implements Repository<ClubSection, string> {
    private readonly storage = new FileStorage<ClubSection>('data/sections');

    async findById(id: string) {
        return this.storage.read(id);
    }

    async findAll(filter?: Partial<ClubSection>) {
        const ids = await this.storage.list();
        const sections = await Promise.all(ids.map(id => this.storage.read(id)));
        const cleaned = sections.filter((s): s is ClubSection => s !== null);

        if (!filter) return cleaned;
        return cleaned.filter(s =>
            Object.entries(filter).every(([k, v]) => (s as any)[k] === v)
        );
    }

    async upsert(section: ClubSection) {
        if (!section.id) {
            const existing = await this.storage.list();
            const maxId = existing
                .map(id => Number(id))
                .reduce((a, b) => (a > b ? a : b), 0);
            section.id = maxId + 1;
        }
        await this.storage.write(String(section.id), section);
        return section;
    }

    async delete(id: string) {
        await this.storage.delete(id);
    }
}
import {Group} from "../models/models";
import {Repository} from "./repository";
import {FileStorage} from "../services/filestorage";

export class GroupRepository implements Repository<Group, string> {
    private readonly storage = new FileStorage<Group>('data/groups');

    async findById(id: string) {
        return this.storage.read(id);
    }

    async findAll(filter?: Partial<Group>) {
        const ids = await this.storage.list();
        const groups = await Promise.all(ids.map(id => this.storage.read(id)));
        const cleaned = groups.filter((g): g is Group => g !== null);

        if (!filter) return cleaned;
        return cleaned.filter(g =>
            Object.entries(filter).every(([k, v]) => (g as any)[k] === v)
        );
    }

    async upsert(group: Group) {
        if (!group.id) {
            const existing = await this.storage.list();
            const maxId = existing
                .map(id => Number(id))
                .reduce((a, b) => (a > b ? a : b), 0);
            group.id = maxId + 1;
        }
        await this.storage.write(String(group.id), group);
        return group;
    }

    async delete(id: string) {
        await this.storage.delete(id);
    }
}
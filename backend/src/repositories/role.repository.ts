import {Repository} from "./repository";
import {Role} from "../models/models";
import {FileStorage} from "../services/filestorage";

export class RoleRepository implements Repository<Role, string> {
    private readonly storage = new FileStorage<Role>('data/roles');

    async findById(id: string) {
        return this.storage.read(id);
    }

    async findAll(filter?: Partial<Role>) {
        const ids = await this.storage.list();
        const roles = await Promise.all(ids.map(id => this.storage.read(id)));
        const cleaned = roles.filter((r): r is Role => r !== null);

        if (!filter) return cleaned;
        return cleaned.filter(r =>
            Object.entries(filter).every(([k, v]) => (r as any)[k] === v)
        );
    }

    async upsert(role: Role) {
        if (!role.id) {
            const existing = await this.storage.list();
            const maxId = existing
                .map(id => Number(id))
                .reduce((a, b) => (a > b ? a : b), 0);
            role.id = maxId + 1;
        }
        await this.storage.write(String(role.id), role);
        return role;
    }

    async delete(id: string) {
        await this.storage.delete(id);
    }
}
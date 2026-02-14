import {Member} from "../models/models.ts";
import {FileStorage} from "../services/filestorage.ts";
import {Repository} from "./repository.ts";

export class MemberRepository implements Repository<Member, string> {

    private readonly storage = new FileStorage<Member>('data/members');

    async findById(id: string) {
        return this.storage.read(id);
    }

    async findAll(filter?: Partial<Member>) {
        const ids = await this.storage.list();
        const members = await Promise.all(ids.map(id => this.storage.read(id)));
        const cleaned = members.filter((m): m is Member => m !== null);
        if (!filter) return cleaned;
        return cleaned.filter(m =>
            Object.entries(filter).every(([key, val]) => (m as any)[key] === val)
        );
    }

    async upsert(member: Member) {
        if (!member.id) {
            const existing = await this.storage.list();
            const maxId = existing
                .map(id => Number(id))
                .reduce((a, b) => (a > b ? a : b), 0);
            member.id = maxId + 1;
        }
        await this.storage.write(String(member.id), member);
        return member;
    }

    async delete(id: string) {
        await this.storage.delete(id);
    }
}
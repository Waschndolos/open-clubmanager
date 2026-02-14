export interface Repository<T, K = string> {
    findById(id: K) : Promise<T|null>;
    findAll(filter?: Partial<T>): Promise<T[]>;
    upsert(item: T): Promise<T>;
    delete(id: K): Promise<void>;
}
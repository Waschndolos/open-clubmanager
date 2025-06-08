export interface NamedEntity {
    id: number;
    name: string;
    description?: string;
}
export interface EntityManagerProps<T extends NamedEntity> {
    title: string;
    fetchFn: () => Promise<T[]>;
    createFn: (data: Omit<T, "id">) => Promise<T>;
    labels?: {
        name?: string;
        description?: string;
        createButton?: string;
    };
}
import { MemberSummary } from "../../api/types";

export interface NamedEntity {
    id: string;
    name: string;
    description?: string;
}
export interface EntityManagerProps<T extends NamedEntity> {
    description: string;
    fetchFn: () => Promise<T[]>;
    createFn: (data: Omit<T, "id">) => Promise<T>;
    updateFn: (data: T) => Promise<T>;
    deleteFn: (data: T) => Promise<void>;
    fetchMembersFn?: (id: string) => Promise<MemberSummary[]>;
    assignMembersFn?: (id: string, memberIds: string[]) => Promise<void>;
    labels?: {
        name?: string;
        description?: string;
        createButton?: string;
    };
}
export interface Id {
    id: string;
}

export interface NamedArtifact extends Id {
    name: string;
    description?: string;
}

export interface MemberContainingNamedArtifact extends NamedArtifact {
    members: Member[];
}

export type MemberSummary = {
    id: string;
    number: number;
    firstName: string;
    lastName: string;
};

export type Member = {
    id: string;
    versionToken?: string;
    number?: number;
    firstName?: string;
    lastName?: string;
    email: string;
    birthday?: Date;
    phone?: string;
    phoneMobile?: string;
    comment?: string;
    entryDate?: Date;
    exitDate?: Date;
    street?: string;
    postalCode?: string;
    city?: string;
    state?: string;
    accountHolder?: string;
    iban?: string;
    bic?: string;
    bankName?: string;
    sepaMandateDate?: Date;
    roles?: Role[];
    groups?: Group[];
    sections?: ClubSection[];
};

export type Group = MemberContainingNamedArtifact

export type Role = MemberContainingNamedArtifact

export type ClubSection = MemberContainingNamedArtifact

export type AppRole = 'ADMIN' | 'TREASURER' | 'SECRETARY' | 'READONLY' | 'TRAINER';

/**
 * Whether the role may change club data in general (members, club structure,
 * inventory, events, households, tasks). Mirrors dataEditors in the server; any
 * other role, such as READONLY or TRAINER (who only records attendance), is
 * read-only.
 */
export function canEditData(role: AppRole | null | undefined): boolean {
    return role === 'ADMIN' || role === 'TREASURER' || role === 'SECRETARY';
}

/** Whether the role may change finance data. Mirrors financeEditors in the server. */
export function canEditFinance(role: AppRole | null | undefined): boolean {
    return role === 'ADMIN' || role === 'TREASURER';
}

export type AppUser = {
    id: string;
    email: string;
    appRole: AppRole;
    createdAt: string;
};

export type FinanceTransaction = {
    id: string;
    date: string;
    description: string;
    amount: number;
    type: 'income' | 'expense';
    category?: string;
    notes?: string;
    createdAt?: string;
    updatedAt?: string;
};

export type MemberFee = {
    id: string;
    memberId: string;
    member?: {
        id: string;
        firstName: string;
        lastName: string;
        number: number;
    };
    amount: number;
    dueDate: string;
    paidDate?: string | null;
    description?: string;
    year: number;
    createdAt?: string;
    updatedAt?: string;
};

export type InventoryItem = {
    id: string;
    name: string;
    description?: string;
    serialNumber?: string;
    category: string;
    quantity: number;
    location: string;
    purchaseDate?: string | null;
    purchasePrice?: number | null;
    createdAt?: string;
    updatedAt?: string;
};

export type InventoryLoan = {
    id: string;
    itemId: string;
    memberId: string;
    loanedAt: string;
    dueDate?: string | null;
    returnedAt?: string | null;
    notes: string;
    createdAt?: string;
};
export interface Id {
    id: number;
}

export interface NamedArtifact extends Id {
    name: string;
    description?: string;
}

export interface MemberContainingNamedArtifact extends NamedArtifact {
    members: Member[];
}

export type MemberSummary = {
    id: number;
    number: number;
    firstName: string;
    lastName: string;
};

export type Member = {
    id: number;
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

export type AppRole = 'ADMIN' | 'TREASURER' | 'SECRETARY' | 'READONLY';

export type AppUser = {
    id: number;
    email: string;
    appRole: AppRole;
    createdAt: string;
};

export type FinanceTransaction = {
    id: number;
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
    id: number;
    memberId: number;
    member?: {
        id: number;
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
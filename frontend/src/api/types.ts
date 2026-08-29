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

export type InventoryItem = {
    id: number;
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
    id: number;
    itemId: number;
    memberId: number;
    loanedAt: string;
    dueDate?: string | null;
    returnedAt?: string | null;
    notes: string;
    createdAt?: string;
};

export type ClubDocument = {
    id: number;
    title: string;
    description?: string;
    category: string;
    filename: string;
    size: number;
    mimeType: string;
    uploadedBy: string;
    createdAt: string;
    updatedAt: string;
};
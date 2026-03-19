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

export type Member = {
    id: number;
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

// ── SQLite storage types ──────────────────────────────────────────────────────

export type WriteLockInfo = {
    holderId: string;
    holderLabel: string;
    acquiredAt: string;
    refreshedAt: string;
    appVersion: string;
};

export type StorageStatus = {
    dataDir: string | null;
    mode: 'edit' | 'readonly';
    lockHolder?: WriteLockInfo;
    lockAgeMs?: number;
};

export type Payment = {
    id: string;
    memberId: string | null;
    amountCents: number;
    currency: string;
    date: string;
    note: string | null;
    createdAt: string;
};

export type Attachment = {
    id: string;
    originalName: string;
    storedRelPath: string;
    mimeType: string;
    sizeBytes: number;
    sha256: string;
    paymentId: string | null;
    memberId: string | null;
    createdAt: string;
};
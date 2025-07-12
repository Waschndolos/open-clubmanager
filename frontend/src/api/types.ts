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

export interface Group extends MemberContainingNamedArtifact {
}

export interface Role extends MemberContainingNamedArtifact {
}

export interface ClubSection extends MemberContainingNamedArtifact{
}
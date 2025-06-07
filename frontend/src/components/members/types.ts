export type Member = {
    id: number;
    number?: number;
    firstName?: string;
    lastName?: string;
    email: string;
    birthday?: string;
    phone?: string;
    phoneMobile?: string;
    comment?: string;
    entryDate?: string; // ISO string, weil aus JSON
    exitDate?: string;
    street?: string;
    postalCode?: string;
    city?: string;
    state?: string;
    accountHolder?: string;
    iban?: string;
    bic?: string;
    bankName?: string;
    sepaMandateDate?: string;
    roles?: Role[];
    groups?: Group[];
    sections?: ClubSection[];
};

export type Role = {
    id: number;
    name: string;
};

export type Group = {
    id: number;
    name: string;
};

export type ClubSection = {
    id: number;
    name: string;
};

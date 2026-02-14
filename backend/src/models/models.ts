export interface BaseEntity {
    id: number;
}


export interface Role extends BaseEntity {
    name: string;
    members?: Member[];
}

export interface Group extends BaseEntity {
    name: string;
    members?: Member[];
}

export interface ClubSection extends BaseEntity {
    name: string;
    members?: Member[];
}


export interface Member extends BaseEntity {
    number: number;

    firstName: string;
    lastName: string;
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
}

export interface UserPreference extends BaseEntity {
    userId: number;
    key: string;
    value: string;
    updatedAt: Date;
}
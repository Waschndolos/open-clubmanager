export interface MemberUpsertInput {
    number: number;
    firstName: string;
    lastName: string;
    email: string;
    birthday?: string | null;
    phone?: string | null;
    phoneMobile?: string | null;
    comment?: string | null;
    entryDate?: string | null;
    exitDate?: string | null;
    street?: string | null;
    postalCode?: string | null;
    city?: string | null;
    state?: string | null;
    accountHolder?: string | null;
    iban?: string | null;
    bic?: string | null;
    bankName?: string | null;
    sepaMandateDate?: string | null;
    roleIds?: number[];
    groupIds?: number[];
    sectionIds?: number[];
    roles?: Array<{ id: number }>;
    groups?: Array<{ id: number }>;
    sections?: Array<{ id: number }>;
}

export interface MembersListQuery {
    page: number;
    pageSize: number;
    search?: string;
}

export interface MembersListResponse<TMember> {
    items: TMember[];
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
}

export interface MemberWithVersion {
    id: number;
    number: number;
    firstName: string;
    lastName: string;
    email: string;
    birthday: Date | null;
    phone: string | null;
    phoneMobile: string | null;
    comment: string | null;
    entryDate: Date | null;
    exitDate: Date | null;
    street: string | null;
    postalCode: string | null;
    city: string | null;
    state: string | null;
    accountHolder: string | null;
    iban: string | null;
    bic: string | null;
    bankName: string | null;
    sepaMandateDate: Date | null;
    roles: Array<{ id: number; name: string }>;
    groups: Array<{ id: number; name: string }>;
    sections: Array<{ id: number; name: string }>;
    versionToken: string;
}


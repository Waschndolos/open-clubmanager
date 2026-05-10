import crypto from 'crypto';

interface HashableMember {
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
    roles: Array<{ id: number }>;
    groups: Array<{ id: number }>;
    sections: Array<{ id: number }>;
}

export function buildMemberVersionToken(member: HashableMember): string {
    const canonical = {
        id: member.id,
        number: member.number,
        firstName: member.firstName,
        lastName: member.lastName,
        email: member.email,
        birthday: member.birthday?.toISOString() ?? null,
        phone: member.phone,
        phoneMobile: member.phoneMobile,
        comment: member.comment,
        entryDate: member.entryDate?.toISOString() ?? null,
        exitDate: member.exitDate?.toISOString() ?? null,
        street: member.street,
        postalCode: member.postalCode,
        city: member.city,
        state: member.state,
        accountHolder: member.accountHolder,
        iban: member.iban,
        bic: member.bic,
        bankName: member.bankName,
        sepaMandateDate: member.sepaMandateDate?.toISOString() ?? null,
        roleIds: member.roles.map((item) => item.id).sort((a, b) => a - b),
        groupIds: member.groups.map((item) => item.id).sort((a, b) => a - b),
        sectionIds: member.sections.map((item) => item.id).sort((a, b) => a - b),
    };

    return crypto
        .createHash('sha256')
        .update(JSON.stringify(canonical), 'utf-8')
        .digest('base64url');
}


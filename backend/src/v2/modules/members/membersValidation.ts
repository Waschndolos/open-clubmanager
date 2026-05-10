import { HttpError } from '../../core/httpError.ts';
import { MembersListQuery, MemberUpsertInput } from './membersTypes.ts';

const MAX_PAGE_SIZE = 100;

function parseDateOrNull(raw: string | null | undefined, fieldName: string): Date | null {
    if (!raw) {
        return null;
    }

    const parsed = new Date(raw);

    if (Number.isNaN(parsed.getTime())) {
        throw new HttpError(400, 'MEMBER_INVALID_DATE', `Field "${fieldName}" contains an invalid date.`);
    }

    return parsed;
}

function normalizeOptionalString(raw: string | null | undefined): string | null {
    if (raw === undefined || raw === null) {
        return null;
    }

    const value = raw.trim();
    return value.length === 0 ? null : value;
}

function normalizeRelationIds(ids: number[] | undefined, relation: Array<{ id: number }> | undefined): number[] {
    const source = ids ?? relation?.map((item) => item.id) ?? [];
    return source
        .map((id) => Number(id))
        .filter((id) => Number.isInteger(id) && id > 0);
}

export function parseMembersListQuery(query: Record<string, unknown>): MembersListQuery {
    const rawPage = Number(query.page ?? 1);
    const rawPageSize = Number(query.pageSize ?? 25);

    const page = Number.isInteger(rawPage) && rawPage > 0 ? rawPage : 1;
    const pageSize =
        Number.isInteger(rawPageSize) && rawPageSize > 0
            ? Math.min(rawPageSize, MAX_PAGE_SIZE)
            : 25;

    const search = typeof query.search === 'string' && query.search.trim().length > 0
        ? query.search.trim()
        : undefined;

    return {
        page,
        pageSize,
        search,
    };
}

export function parseMemberUpsertInput(input: Partial<MemberUpsertInput>): {
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
    roleIds: number[];
    groupIds: number[];
    sectionIds: number[];
} {
    const number = Number(input.number);
    const firstName = String(input.firstName ?? '').trim();
    const lastName = String(input.lastName ?? '').trim();
    const email = String(input.email ?? '').trim().toLowerCase();

    if (!Number.isInteger(number) || number <= 0) {
        throw new HttpError(400, 'MEMBER_INVALID_NUMBER', 'Member number must be a positive integer.');
    }

    if (!firstName) {
        throw new HttpError(400, 'MEMBER_INVALID_FIRST_NAME', 'First name is required.');
    }

    if (!lastName) {
        throw new HttpError(400, 'MEMBER_INVALID_LAST_NAME', 'Last name is required.');
    }

    if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
        throw new HttpError(400, 'MEMBER_INVALID_EMAIL', 'Email is required and must be valid.');
    }

    return {
        number,
        firstName,
        lastName,
        email,
        birthday: parseDateOrNull(input.birthday, 'birthday'),
        phone: normalizeOptionalString(input.phone),
        phoneMobile: normalizeOptionalString(input.phoneMobile),
        comment: normalizeOptionalString(input.comment),
        entryDate: parseDateOrNull(input.entryDate, 'entryDate'),
        exitDate: parseDateOrNull(input.exitDate, 'exitDate'),
        street: normalizeOptionalString(input.street),
        postalCode: normalizeOptionalString(input.postalCode),
        city: normalizeOptionalString(input.city),
        state: normalizeOptionalString(input.state),
        accountHolder: normalizeOptionalString(input.accountHolder),
        iban: normalizeOptionalString(input.iban),
        bic: normalizeOptionalString(input.bic),
        bankName: normalizeOptionalString(input.bankName),
        sepaMandateDate: parseDateOrNull(input.sepaMandateDate, 'sepaMandateDate'),
        roleIds: normalizeRelationIds(input.roleIds, input.roles),
        groupIds: normalizeRelationIds(input.groupIds, input.groups),
        sectionIds: normalizeRelationIds(input.sectionIds, input.sections),
    };
}


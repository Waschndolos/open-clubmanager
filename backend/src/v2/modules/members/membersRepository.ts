import { getClient } from '../../../db.ts';

const memberInclude = {
    roles: true,
    groups: true,
    sections: true,
} as const;

type NamedEntity = {
    id: number;
    name: string;
};

export interface PersistedMember {
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
    roles: NamedEntity[];
    groups: NamedEntity[];
    sections: NamedEntity[];
}

export interface MemberPersistenceInput {
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
}

export interface MembersRepository {
    list(args: { skip: number; take: number; search?: string }): Promise<PersistedMember[]>;
    count(search?: string): Promise<number>;
    findById(id: number): Promise<PersistedMember | null>;
    create(data: MemberPersistenceInput): Promise<PersistedMember>;
    update(id: number, data: MemberPersistenceInput): Promise<PersistedMember>;
    delete(id: number): Promise<void>;
}

type MembersDbClient = {
    member: {
        findMany(args: Record<string, unknown>): Promise<PersistedMember[]>;
        count(args: Record<string, unknown>): Promise<number>;
        findUnique(args: Record<string, unknown>): Promise<PersistedMember | null>;
        create(args: Record<string, unknown>): Promise<PersistedMember>;
        update(args: Record<string, unknown>): Promise<PersistedMember>;
        delete(args: Record<string, unknown>): Promise<void>;
    };
};

function buildSearchWhere(search?: string): Record<string, unknown> | undefined {
    if (!search) {
        return undefined;
    }

    const asNumber = Number(search);

    const searchFilters: Record<string, unknown>[] = [
        { firstName: { contains: search } },
        { lastName: { contains: search } },
        { email: { contains: search } },
    ];

    if (Number.isInteger(asNumber)) {
        searchFilters.push({ number: asNumber });
    }

    return {
        OR: searchFilters,
    };
}

export class PrismaMembersRepository implements MembersRepository {
    async list(args: { skip: number; take: number; search?: string }): Promise<PersistedMember[]> {
        const prisma = (await getClient()) as unknown as MembersDbClient;
        return prisma.member.findMany({
            skip: args.skip,
            take: args.take,
            where: buildSearchWhere(args.search),
            orderBy: [{ lastName: 'asc' }, { firstName: 'asc' }, { id: 'asc' }],
            include: memberInclude,
        });
    }

    async count(search?: string): Promise<number> {
        const prisma = (await getClient()) as unknown as MembersDbClient;
        return prisma.member.count({
            where: buildSearchWhere(search),
        });
    }

    async findById(id: number): Promise<PersistedMember | null> {
        const prisma = (await getClient()) as unknown as MembersDbClient;
        return prisma.member.findUnique({
            where: { id },
            include: memberInclude,
        });
    }

    async create(data: MemberPersistenceInput): Promise<PersistedMember> {
        const prisma = (await getClient()) as unknown as MembersDbClient;
        return prisma.member.create({
            data: {
                number: data.number,
                firstName: data.firstName,
                lastName: data.lastName,
                email: data.email,
                birthday: data.birthday,
                phone: data.phone,
                phoneMobile: data.phoneMobile,
                comment: data.comment,
                entryDate: data.entryDate,
                exitDate: data.exitDate,
                street: data.street,
                postalCode: data.postalCode,
                city: data.city,
                state: data.state,
                accountHolder: data.accountHolder,
                iban: data.iban,
                bic: data.bic,
                bankName: data.bankName,
                sepaMandateDate: data.sepaMandateDate,
                roles: { connect: data.roleIds.map((id) => ({ id })) },
                groups: { connect: data.groupIds.map((id) => ({ id })) },
                sections: { connect: data.sectionIds.map((id) => ({ id })) },
            },
            include: memberInclude,
        });
    }

    async update(id: number, data: MemberPersistenceInput): Promise<PersistedMember> {
        const prisma = (await getClient()) as unknown as MembersDbClient;
        return prisma.member.update({
            where: { id },
            data: {
                number: data.number,
                firstName: data.firstName,
                lastName: data.lastName,
                email: data.email,
                birthday: data.birthday,
                phone: data.phone,
                phoneMobile: data.phoneMobile,
                comment: data.comment,
                entryDate: data.entryDate,
                exitDate: data.exitDate,
                street: data.street,
                postalCode: data.postalCode,
                city: data.city,
                state: data.state,
                accountHolder: data.accountHolder,
                iban: data.iban,
                bic: data.bic,
                bankName: data.bankName,
                sepaMandateDate: data.sepaMandateDate,
                roles: { set: data.roleIds.map((itemId) => ({ id: itemId })) },
                groups: { set: data.groupIds.map((itemId) => ({ id: itemId })) },
                sections: { set: data.sectionIds.map((itemId) => ({ id: itemId })) },
            },
            include: memberInclude,
        });
    }

    async delete(id: number): Promise<void> {
        const prisma = (await getClient()) as unknown as MembersDbClient;
        await prisma.member.delete({ where: { id } });
    }
}


import { HttpError } from '../../core/httpError.ts';
import {
    MemberWithVersion,
    MembersListQuery,
    MembersListResponse,
    MemberUpsertInput,
} from './membersTypes.ts';
import {
    MemberPersistenceInput,
    MembersRepository,
    PersistedMember,
} from './membersRepository.ts';
import { parseMemberUpsertInput } from './membersValidation.ts';
import { buildMemberVersionToken } from './memberVersion.ts';

function mapMember(member: PersistedMember): MemberWithVersion {
    return {
        id: member.id,
        number: member.number,
        firstName: member.firstName,
        lastName: member.lastName,
        email: member.email,
        birthday: member.birthday,
        phone: member.phone,
        phoneMobile: member.phoneMobile,
        comment: member.comment,
        entryDate: member.entryDate,
        exitDate: member.exitDate,
        street: member.street,
        postalCode: member.postalCode,
        city: member.city,
        state: member.state,
        accountHolder: member.accountHolder,
        iban: member.iban,
        bic: member.bic,
        bankName: member.bankName,
        sepaMandateDate: member.sepaMandateDate,
        roles: member.roles,
        groups: member.groups,
        sections: member.sections,
        versionToken: buildMemberVersionToken(member),
    };
}

function mapPrismaError(err: unknown): never {
    const maybeCode = (err as { code?: string } | undefined)?.code;

    if (maybeCode === 'P2002') {
        throw new HttpError(409, 'MEMBER_CONFLICT', 'A member with the same unique value already exists.');
    }

    if (maybeCode === 'P2025') {
        throw new HttpError(404, 'MEMBER_NOT_FOUND', 'Member was not found.');
    }

    throw err;
}

export class MembersService {
    constructor(private readonly membersRepository: MembersRepository) {}

    async list(query: MembersListQuery): Promise<MembersListResponse<MemberWithVersion>> {
        const skip = (query.page - 1) * query.pageSize;
        const [items, total] = await Promise.all([
            this.membersRepository.list({
                skip,
                take: query.pageSize,
                search: query.search,
            }),
            this.membersRepository.count(query.search),
        ]);

        const totalPages = Math.max(1, Math.ceil(total / query.pageSize));

        return {
            items: items.map(mapMember),
            page: query.page,
            pageSize: query.pageSize,
            total,
            totalPages,
        };
    }

    async getById(id: number): Promise<MemberWithVersion> {
        const member = await this.membersRepository.findById(id);

        if (!member) {
            throw new HttpError(404, 'MEMBER_NOT_FOUND', 'Member was not found.');
        }

        return mapMember(member);
    }

    async create(input: Partial<MemberUpsertInput>): Promise<MemberWithVersion> {
        const parsed: MemberPersistenceInput = parseMemberUpsertInput(input);

        try {
            const created = await this.membersRepository.create(parsed);
            return mapMember(created);
        } catch (err) {
            mapPrismaError(err);
        }
    }

    async update(id: number, input: Partial<MemberUpsertInput>, expectedVersionToken?: string): Promise<MemberWithVersion> {
        if (!expectedVersionToken) {
            throw new HttpError(400, 'MEMBER_MISSING_VERSION', 'expectedVersionToken is required for updates.');
        }

        const existing = await this.membersRepository.findById(id);

        if (!existing) {
            throw new HttpError(404, 'MEMBER_NOT_FOUND', 'Member was not found.');
        }

        const currentVersionToken = buildMemberVersionToken(existing);
        if (expectedVersionToken !== currentVersionToken) {
            throw new HttpError(
                409,
                'MEMBER_VERSION_CONFLICT',
                'The member changed since you last loaded it. Please reload and try again.'
            );
        }

        const parsed: MemberPersistenceInput = parseMemberUpsertInput(input);

        try {
            const updated = await this.membersRepository.update(id, parsed);
            return mapMember(updated);
        } catch (err) {
            mapPrismaError(err);
        }
    }

    async delete(id: number): Promise<void> {
        try {
            await this.membersRepository.delete(id);
        } catch (err) {
            mapPrismaError(err);
        }
    }
}


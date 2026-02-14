import { MemberRepository } from '../repositories/member.repository';
import { GroupRepository } from '../repositories/group.repository';
import { ClubSectionRepository } from '../repositories/clubSection.repository';

const memberRepo = new MemberRepository();
const groupRepo = new GroupRepository();
const sectionRepo = new ClubSectionRepository();

/** Helper – calculate age from a birthdate */
function calcAge(birthday: Date | undefined): number | null {
    if (!birthday) return null;
    const diff = Date.now() - birthday.getTime();
    return Math.floor(diff / (1000 * 60 * 60 * 24 * 365.25));
}

/**
 * Returns a summary object used by the dashboard.
 * Example shape:
 * {
 *   totalMembers: 124,
 *   activeMembers: 98,
 *   averageAge: 34.2,
 *   membersPerGroup: { "Junior": 45, "Senior": 53 },
 *   membersPerSection: { "Football": 70, "Tennis": 30 }
 * }
 */
export async function getDashboardStats() {
    // ----- MEMBERS -----
    const allMembers = await memberRepo.findAll();
    const totalMembers = allMembers.length;
    const activeMembers = allMembers.filter(m => !m.exitDate).length;

    const ages = allMembers
        .map(m => calcAge(m.birthday ?? undefined))
        .filter((a): a is number => a !== null);
    const averageAge = ages.length ? ages.reduce((a, b) => a + b, 0) / ages.length : null;

    // ----- GROUPS -----
    const groups = await groupRepo.findAll();
    const membersPerGroup: Record<string, number> = {};
    for (const g of groups) {
        membersPerGroup[g.name] = g.members?.length ?? 0;
    }

    // ----- SECTIONS -----
    const sections = await sectionRepo.findAll();
    const membersPerSection: Record<string, number> = {};
    for (const s of sections) {
        membersPerSection[s.name] = s.members?.length ?? 0;
    }

    return {
        totalMembers,
        activeMembers,
        averageAge,
        membersPerGroup,
        membersPerSection,
    };
}

/**
 * Returns a breakdown of members by role.
 * Example:
 * {
 *   Trainer: 12,
 *   Player: 85,
 *   BoardMember: 3
 * }
 */
export async function getMemberCountByRole() {
    const allMembers = await memberRepo.findAll();
    const roleMap: Record<string, number> = {};

    for (const m of allMembers) {
        if (!m.roles) continue;
        for (const r of m.roles) {
            roleMap[r.name] = (roleMap[r.name] ?? 0) + 1;
        }
    }

    return roleMap;
}

/**
 * Example of a time‑series statistic – members joined per month for the
 * last 12 months.
 */
export async function getMonthlyJoinTrend() {
    const now = new Date();
    const twelveMonthsAgo = new Date(now.getFullYear(), now.getMonth() - 11, 1);

    const allMembers = await memberRepo.findAll();
    const trend: Record<string, number> = {};

    for (let i = 0; i < 12; i++) {
        const month = new Date(twelveMonthsAgo.getFullYear(), twelveMonthsAgo.getMonth() + i, 1);
        const key = month.toISOString().slice(0, 7); // "YYYY-MM"
        trend[key] = 0;
    }

    for (const m of allMembers) {
        if (!m.entryDate) continue;
        if (m.entryDate < twelveMonthsAgo) continue;
        const key = m.entryDate.toISOString().slice(0, 7);
        if (trend[key] !== undefined) trend[key] += 1;
    }

    return trend;
}
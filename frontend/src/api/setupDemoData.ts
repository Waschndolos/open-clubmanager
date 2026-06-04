import { createGroup } from './groups';
import { createMember } from './members';
import { createRole } from './roles';
import { createSection } from './sections';
import { createMemberFee, createTransaction } from './finance';

export interface SetupDemoDataResult {
    roles: number;
    groups: number;
    sections: number;
    members: number;
    transactions: number;
    memberFees: number;
}

export async function seedSetupDemoData(): Promise<SetupDemoDataResult> {
    const roleNames = ['Vorstand', 'Trainer', 'Kassenwart'];
    const groupNames = ['Jugend', 'Erwachsene'];
    const sectionNames = ['Fussball', 'Turnen'];

    await Promise.all(roleNames.map((name) => createRole({ name, members: [] })));
    await Promise.all(groupNames.map((name) => createGroup({ name, members: [] })));
    await Promise.all(sectionNames.map((name) => createSection({ name, members: [] })));

    const sampleMembers = [
        { number: 1001, firstName: 'Anna', lastName: 'Mueller', email: 'anna.mueller@example.org' },
        { number: 1002, firstName: 'Ben', lastName: 'Schmidt', email: 'ben.schmidt@example.org' },
        { number: 1003, firstName: 'Clara', lastName: 'Fischer', email: 'clara.fischer@example.org' },
    ];

    const createdMembers = [] as Array<{ id: number }>;
    for (const member of sampleMembers) {
        const created = await createMember(member);
        createdMembers.push({ id: created.id });
    }

    await createTransaction({
        date: new Date('2026-01-15T10:00:00.000Z').toISOString(),
        description: 'Mitgliedsbeitraege Januar',
        amount: 180,
        type: 'income',
        category: 'Beitraege',
    });
    await createTransaction({
        date: new Date('2026-01-20T09:30:00.000Z').toISOString(),
        description: 'Sportgeraete Einkauf',
        amount: 95,
        type: 'expense',
        category: 'Material',
    });

    const currentYear = new Date().getUTCFullYear();
    for (const member of createdMembers) {
        await createMemberFee({
            memberId: member.id,
            amount: 60,
            dueDate: new Date(`${currentYear}-03-01T00:00:00.000Z`).toISOString(),
            year: currentYear,
            description: 'Jahresbeitrag',
        });
    }

    return {
        roles: roleNames.length,
        groups: groupNames.length,
        sections: sectionNames.length,
        members: createdMembers.length,
        transactions: 2,
        memberFees: createdMembers.length,
    };
}


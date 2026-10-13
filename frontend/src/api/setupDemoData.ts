import { assignGroupMembers, createGroup } from './groups';
import { createMember } from './members';
import { assignRoleMembers, createRole } from './roles';
import { assignSectionMembers, createSection } from './sections';
import { createMemberFee, createTransaction } from './finance';
import { createInventoryItem, createInventoryLoan } from './inventory';
import { recordMaintenance, setItemStatus } from './maintenance';
import { addDays, createEvent, localTimeZone, ymd } from './events';
import { saveAttendance } from './attendance';
import { Member } from './types';

export interface SetupDemoDataResult {
    roles: number;
    groups: number;
    sections: number;
    members: number;
    transactions: number;
    memberFees: number;
    inventoryItems: number;
    inventoryLoans: number;
}

// Demo club: "Kiteclub Nordwind e.V.", a kitesurf club on the Baltic Sea coast.

const ROLES = ['Vorstand', 'Kassenwart', 'Schriftführer', 'Kitelehrer (VDWS)', 'Rettungsschwimmer', 'Materialwart'];
const GROUPS = ['Jugend (U18)', 'Erwachsene', 'Familienmitgliedschaft', 'Fördermitglieder'];
const SECTIONS = ['Twintip & Freestyle', 'Wave & Strapless', 'Foil', 'Snowkiten', 'Kiteschule'];

type Fee = 'adult' | 'youth' | 'family' | 'supporter';

const FEE_AMOUNTS: Record<Fee, number> = {
    adult: 180,
    youth: 90,
    family: 120,
    supporter: 50,
};

const FEE_GROUP: Record<Fee, string> = {
    adult: 'Erwachsene',
    youth: 'Jugend (U18)',
    family: 'Familienmitgliedschaft',
    supporter: 'Fördermitglieder',
};

interface DemoMember {
    data: Omit<Member, 'id'>;
    fee: Fee;
    roles?: string[];
    sections?: string[];
}

// Standard documentation IBAN, never a real account.
const DEMO_IBAN = 'DE89370400440532013000';
const DEMO_BIC = 'COBADEFFXXX';

function date(value: string): Date {
    return new Date(`${value}T00:00:00.000Z`);
}

function sepa(accountHolder: string, mandateDate: string) {
    return {
        accountHolder,
        iban: DEMO_IBAN,
        bic: DEMO_BIC,
        bankName: 'Demo Bank',
        sepaMandateDate: date(mandateDate),
    };
}

const MEMBERS: DemoMember[] = [
    {
        data: {
            number: 1001, firstName: 'Jannik', lastName: 'Petersen', email: 'jannik.petersen@example.org',
            birthday: date('1984-05-12'), entryDate: date('2012-04-01'), phoneMobile: '+49 170 1000001',
            street: 'Strandweg 3', postalCode: '23769', city: 'Fehmarn', state: 'Schleswig-Holstein',
            comment: 'Gründungsmitglied, 1. Vorsitzender',
            ...sepa('Jannik Petersen', '2012-04-01'),
        },
        fee: 'adult', roles: ['Vorstand'], sections: ['Wave & Strapless'],
    },
    {
        data: {
            number: 1002, firstName: 'Merle', lastName: 'Hansen', email: 'merle.hansen@example.org',
            birthday: date('1990-09-23'), entryDate: date('2014-05-15'), phoneMobile: '+49 170 1000002',
            street: 'Deichstraße 17', postalCode: '24321', city: 'Hohwacht', state: 'Schleswig-Holstein',
            ...sepa('Merle Hansen', '2014-05-15'),
        },
        fee: 'adult', roles: ['Vorstand', 'Kassenwart'], sections: ['Foil'],
    },
    {
        data: {
            number: 1003, firstName: 'Ole', lastName: 'Jensen', email: 'ole.jensen@example.org',
            birthday: date('1979-02-08'), entryDate: date('2013-03-01'), phoneMobile: '+49 170 1000003',
            street: 'Am Hafen 5', postalCode: '23774', city: 'Heiligenhafen', state: 'Schleswig-Holstein',
            comment: 'VDWS-Lizenz bis 12/2027',
        },
        fee: 'adult', roles: ['Kitelehrer (VDWS)', 'Materialwart'], sections: ['Kiteschule', 'Twintip & Freestyle'],
    },
    {
        data: {
            number: 1004, firstName: 'Lea', lastName: 'Brandt', email: 'lea.brandt@example.org',
            birthday: date('1995-11-30'), entryDate: date('2018-06-01'), phoneMobile: '+49 170 1000004',
            street: 'Möwenring 22', postalCode: '23747', city: 'Dahme', state: 'Schleswig-Holstein',
            comment: 'Rettungsschwimmerin DLRG Silber',
        },
        fee: 'adult', roles: ['Kitelehrer (VDWS)', 'Rettungsschwimmer'], sections: ['Kiteschule', 'Wave & Strapless'],
    },
    {
        data: {
            number: 1005, firstName: 'Finn', lastName: 'Clausen', email: 'finn.clausen@example.org',
            birthday: date('1988-07-04'), entryDate: date('2016-04-10'),
            street: 'Lindenallee 9', postalCode: '24103', city: 'Kiel', state: 'Schleswig-Holstein',
        },
        fee: 'adult', roles: ['Schriftführer'], sections: ['Twintip & Freestyle', 'Snowkiten'],
    },
    {
        data: {
            number: 1006, firstName: 'Sophie', lastName: 'Lorenzen', email: 'sophie.lorenzen@example.org',
            birthday: date('1992-03-17'), entryDate: date('2019-05-01'), phoneMobile: '+49 170 1000006',
            street: 'Kurpromenade 41', postalCode: '23683', city: 'Scharbeutz', state: 'Schleswig-Holstein',
            ...sepa('Sophie Lorenzen', '2019-05-01'),
        },
        fee: 'adult', sections: ['Foil'],
    },
    {
        data: {
            number: 1007, firstName: 'Mats', lastName: 'Andresen', email: 'mats.andresen@example.org',
            birthday: date('1986-12-01'), entryDate: date('2015-07-01'),
            street: 'Dünenpfad 2', postalCode: '25980', city: 'Sylt', state: 'Schleswig-Holstein',
            comment: 'Fährt im Winter Snowkite-Ausfahrten nach Norwegen',
        },
        fee: 'adult', sections: ['Wave & Strapless', 'Snowkiten'],
    },
    {
        data: {
            number: 1008, firstName: 'Kira', lastName: 'Thomsen', email: 'kira.thomsen@example.org',
            birthday: date('2001-08-19'), entryDate: date('2022-04-23'), phoneMobile: '+49 170 1000008',
            street: 'Fischerstraße 14', postalCode: '23730', city: 'Neustadt in Holstein', state: 'Schleswig-Holstein',
            ...sepa('Kira Thomsen', '2022-04-23'),
        },
        fee: 'adult', sections: ['Twintip & Freestyle'],
    },
    {
        data: {
            number: 1009, firstName: 'Henrik', lastName: 'Nissen', email: 'henrik.nissen@example.org',
            birthday: date('1975-01-26'), entryDate: date('2020-06-01'),
            street: 'Bergstraße 7', postalCode: '20095', city: 'Hamburg', state: 'Hamburg',
            comment: 'Unterstützt den Verein als Fördermitglied',
        },
        fee: 'supporter',
    },
    {
        data: {
            number: 1010, firstName: 'Nele', lastName: 'Kruse', email: 'nele.kruse@example.org',
            birthday: date('2009-04-14'), entryDate: date('2023-05-06'),
            street: 'Strandweg 11', postalCode: '23769', city: 'Fehmarn', state: 'Schleswig-Holstein',
            comment: 'Einverständniserklärung der Eltern liegt vor',
        },
        fee: 'youth', sections: ['Kiteschule', 'Twintip & Freestyle'],
    },
    {
        data: {
            number: 1011, firstName: 'Paul', lastName: 'Kruse', email: 'paul.kruse@example.org',
            birthday: date('2011-10-02'), entryDate: date('2024-04-20'),
            street: 'Strandweg 11', postalCode: '23769', city: 'Fehmarn', state: 'Schleswig-Holstein',
            comment: 'Geschwisterkind von Nele Kruse',
        },
        fee: 'family', sections: ['Kiteschule'],
    },
    {
        data: {
            number: 1012, firstName: 'Tjark', lastName: 'Albers', email: 'tjark.albers@example.org',
            birthday: date('2008-06-28'), entryDate: date('2022-06-15'),
            street: 'Ostseeallee 30', postalCode: '23570', city: 'Lübeck-Travemünde', state: 'Schleswig-Holstein',
        },
        fee: 'youth', sections: ['Twintip & Freestyle', 'Foil'],
    },
    {
        data: {
            number: 1013, firstName: 'Ida', lastName: 'Wendt', email: 'ida.wendt@example.org',
            birthday: date('1998-02-11'), entryDate: date('2021-05-01'),
            street: 'Rosenweg 4', postalCode: '18119', city: 'Rostock-Warnemünde', state: 'Mecklenburg-Vorpommern',
        },
        fee: 'adult', sections: ['Wave & Strapless'],
    },
    {
        data: {
            number: 1014, firstName: 'Lasse', lastName: 'Böttcher', email: 'lasse.boettcher@example.org',
            birthday: date('1993-09-05'), entryDate: date('2017-04-01'), exitDate: date('2025-12-31'),
            street: 'Wiesengrund 18', postalCode: '24937', city: 'Flensburg', state: 'Schleswig-Holstein',
            comment: 'Ausgetreten wegen Umzug nach Portugal',
        },
        fee: 'adult', sections: ['Wave & Strapless'],
    },
    {
        data: {
            number: 1015, firstName: 'Greta', lastName: 'Möller', email: 'greta.moeller@example.org',
            birthday: date('1987-05-20'), entryDate: date(`${new Date().getUTCFullYear()}-05-01`),
            street: 'Seestraße 25', postalCode: '23769', city: 'Fehmarn', state: 'Schleswig-Holstein',
            comment: 'Neumitglied nach Grundkurs in der Vereins-Kiteschule',
            ...sepa('Greta Möller', `${new Date().getUTCFullYear()}-05-01`),
        },
        fee: 'adult', sections: ['Kiteschule'],
    },
];

interface DemoTransaction {
    month: string; // MM-DD within the current year
    description: string;
    amount: number;
    type: 'income' | 'expense';
    category: string;
}

const TRANSACTIONS: DemoTransaction[] = [
    { month: '01-15', description: 'Mitgliedsbeiträge SEPA-Lastschrift Januar', amount: 1530, type: 'income', category: 'Beiträge' },
    { month: '02-03', description: 'Haftpflichtversicherung Verein (Jahresbeitrag)', amount: 420, type: 'expense', category: 'Versicherung' },
    { month: '02-20', description: 'Pacht Vereinsgelände Strandabschnitt Gold', amount: 1200, type: 'expense', category: 'Pacht' },
    { month: '03-10', description: 'Neue Schulungskites 7 m / 9 m / 12 m', amount: 2850, type: 'expense', category: 'Material' },
    { month: '03-28', description: 'Zuschuss Landessportbund Jugendarbeit', amount: 800, type: 'income', category: 'Zuschüsse' },
    { month: '04-12', description: 'Wartung Rettungsboot und Außenborder', amount: 340, type: 'expense', category: 'Sicherheit' },
    { month: '05-18', description: 'Kursgebühren Kiteschule Mai', amount: 2140, type: 'income', category: 'Kiteschule' },
    { month: '06-07', description: 'Startgelder Fehmarn Kite Cup', amount: 960, type: 'income', category: 'Veranstaltungen' },
    { month: '06-08', description: 'Pokale und Verpflegung Fehmarn Kite Cup', amount: 610, type: 'expense', category: 'Veranstaltungen' },
    { month: '07-14', description: 'Kursgebühren Kiteschule Juli', amount: 3260, type: 'income', category: 'Kiteschule' },
    { month: '07-22', description: 'Ersatz-Bladder und Reparaturmaterial', amount: 185, type: 'expense', category: 'Material' },
    { month: '08-30', description: 'Spende Surfshop Nordwind', amount: 500, type: 'income', category: 'Spenden' },
    { month: '09-05', description: 'VDWS-Lizenzverlängerung Kitelehrer', amount: 270, type: 'expense', category: 'Ausbildung' },
];

const INVENTORY = [
    { key: 'kite9', name: 'Schulungskite 9 m', category: 'Kites', quantity: 4, location: 'Containerlager Strand', purchasePrice: 950, serialNumber: 'NW-K9' },
    { key: 'kite12', name: 'Schulungskite 12 m', category: 'Kites', quantity: 3, location: 'Containerlager Strand', purchasePrice: 1090, serialNumber: 'NW-K12' },
    { key: 'bar', name: 'Control Bar 45 m Leinen', category: 'Bars', quantity: 6, location: 'Containerlager Strand', purchasePrice: 420 },
    { key: 'twintip', name: 'Twintip-Board 138 cm', category: 'Boards', quantity: 5, location: 'Boardständer Vereinsheim', purchasePrice: 520 },
    { key: 'foil', name: 'Foilboard mit Hydrofoil-Set', category: 'Boards', quantity: 2, location: 'Vereinsheim Werkstatt', purchasePrice: 1650, serialNumber: 'NW-F01' },
    { key: 'harness', name: 'Hüfttrapez (Größe M/L)', category: 'Trapeze', quantity: 8, location: 'Umkleide Vereinsheim', purchasePrice: 180 },
    { key: 'wetsuit', name: 'Neoprenanzug 5/4 mm', category: 'Neopren', quantity: 10, location: 'Umkleide Vereinsheim', purchasePrice: 260 },
    { key: 'helmet', name: 'Wassersport-Helm', category: 'Sicherheit', quantity: 12, location: 'Umkleide Vereinsheim', purchasePrice: 65 },
    { key: 'boat', name: 'Rettungsboot (Schlauchboot, 15 PS)', category: 'Sicherheit', quantity: 1, location: 'Slipanlage', purchasePrice: 7800, serialNumber: 'NW-RB-01' },
    { key: 'radio', name: 'Funkgerät / Headset-Set für Kiteschule', category: 'Kiteschule', quantity: 4, location: 'Vereinsheim Büro', purchasePrice: 140 },
] as const;

export async function seedSetupDemoData(): Promise<SetupDemoDataResult> {
    const currentYear = new Date().getUTCFullYear();

    const [roles, groups, sections] = await Promise.all([
        Promise.all(ROLES.map((name) => createRole({ name, members: [] }))),
        Promise.all(GROUPS.map((name) => createGroup({ name, members: [] }))),
        Promise.all(SECTIONS.map((name) => createSection({ name, members: [] }))),
    ]);
    const roleIds = new Map(roles.map((r) => [r.name, r.id]));
    const groupIds = new Map(groups.map((g) => [g.name, g.id]));
    const sectionIds = new Map(sections.map((s) => [s.name, s.id]));

    const roleMembers = new Map<string, string[]>();
    const groupMembers = new Map<string, string[]>();
    const sectionMembers = new Map<string, string[]>();
    const push = (map: Map<string, string[]>, key: string, id: string) => {
        map.set(key, [...(map.get(key) ?? []), id]);
    };

    const createdMembers: Array<{ id: string; demo: DemoMember }> = [];
    for (const demo of MEMBERS) {
        const created = await createMember(demo.data);
        createdMembers.push({ id: created.id, demo });
        push(groupMembers, FEE_GROUP[demo.fee], created.id);
        demo.roles?.forEach((role) => push(roleMembers, role, created.id));
        demo.sections?.forEach((section) => push(sectionMembers, section, created.id));
    }

    await Promise.all([
        ...[...roleMembers].map(([name, ids]) => assignRoleMembers(roleIds.get(name)!, ids)),
        ...[...groupMembers].map(([name, ids]) => assignGroupMembers(groupIds.get(name)!, ids)),
        ...[...sectionMembers].map(([name, ids]) => assignSectionMembers(sectionIds.get(name)!, ids)),
    ]);

    for (const tx of TRANSACTIONS) {
        await createTransaction({
            date: new Date(`${currentYear}-${tx.month}T10:00:00.000Z`).toISOString(),
            description: tx.description,
            amount: tx.amount,
            type: tx.type,
            category: tx.category,
        });
    }

    // Annual fee for every active member; most have paid via SEPA, a few are still open.
    const unpaid = new Set([1007, 1012, 1013]);
    let memberFees = 0;
    for (const { id, demo } of createdMembers) {
        if (demo.data.exitDate) continue;
        await createMemberFee({
            memberId: id,
            amount: FEE_AMOUNTS[demo.fee],
            dueDate: new Date(`${currentYear}-03-01T00:00:00.000Z`).toISOString(),
            paidDate: unpaid.has(demo.data.number!)
                ? null
                : new Date(`${currentYear}-01-15T00:00:00.000Z`).toISOString(),
            year: currentYear,
            description: `Jahresbeitrag ${currentYear}`,
        });
        memberFees++;
    }

    const itemIds = new Map<string, string>();
    for (const { key, ...item } of INVENTORY) {
        const created = await createInventoryItem({
            ...item,
            purchaseDate: new Date(`${currentYear}-03-10T00:00:00.000Z`).toISOString(),
        });
        itemIds.set(key, created.id);
    }

    // Inspection schedules: the rescue boat is overdue, the helmets are due
    // soon, one kite is damaged and the radios are in repair (not lendable).
    const day = (offset: number) => {
        const d = new Date();
        d.setUTCDate(d.getUTCDate() + offset);
        return d.toISOString().slice(0, 10);
    };
    const yearAgo = (offset: number) => {
        const d = new Date(`${day(offset)}T00:00:00Z`);
        d.setUTCFullYear(d.getUTCFullYear() - 1);
        return d.toISOString().slice(0, 10);
    };
    await setItemStatus(itemIds.get('boat')!, { condition: 'ready', intervalMonths: 12 });
    await recordMaintenance(itemIds.get('boat')!, {
        date: yearAgo(-12), performedBy: 'Bootsservice Fehmarn', note: 'Motorinspektion und Schlauchprüfung', cost: 320,
    });
    await setItemStatus(itemIds.get('helmet')!, { condition: 'ready', intervalMonths: 24, nextDue: day(14) });
    await setItemStatus(itemIds.get('kite9')!, { condition: 'ready', intervalMonths: 12, nextDue: day(120) });
    await setItemStatus(itemIds.get('kite12')!, { condition: 'damaged' });
    await recordMaintenance(itemIds.get('kite12')!, {
        date: day(-5), performedBy: 'Kiteschule', note: 'Riss in der Leading Edge festgestellt',
    });
    await setItemStatus(itemIds.get('radio')!, { condition: 'in_repair' });

    const memberByNumber = new Map(createdMembers.map(({ id, demo }) => [demo.data.number, id]));
    const loans = [
        { item: 'foil', member: 1006, notes: 'Foil-Training für die Vereinsmeisterschaft' },
        { item: 'kite9', member: 1010, notes: 'Jugendtraining am Wochenende' },
        { item: 'wetsuit', member: 1015, notes: 'Leihneo bis eigener Anzug geliefert wird' },
    ];
    for (const loan of loans) {
        await createInventoryLoan({
            itemId: itemIds.get(loan.item)!,
            memberId: memberByNumber.get(loan.member)!,
            loanedAt: new Date(`${currentYear}-08-01T09:00:00.000Z`).toISOString(),
            dueDate: new Date(`${currentYear}-08-15T18:00:00.000Z`).toISOString(),
            notes: loan.notes,
        });
    }

    // Club calendar: weekly youth training, a monthly board meeting, a
    // competition weekend and the summer party, relative to today.
    const at = (offsetDays: number, hour: number, minute = 0) => {
        const d = new Date();
        d.setDate(d.getDate() + offsetDays);
        d.setHours(hour, minute, 0, 0);
        return d;
    };
    const nextWeekday = (weekday: number) => (weekday - new Date().getDay() + 7) % 7;
    const tz = localTimeZone();
    const events = [
        {
            title: 'Jugendtraining Twintip', category: 'training' as const, location: 'Kitespot Nordstrand',
            start: at(nextWeekday(2) - 14, 17, 30), minutes: 90, recurrence: 'weekly' as const,
            groupIds: [groupIds.get('Jugend (U18)')!], sectionIds: [sectionIds.get('Twintip & Freestyle')!],
        },
        {
            title: 'Foil-Treff', category: 'training' as const, location: 'Vereinsheim Slipanlage',
            start: at(nextWeekday(6) - 7, 10, 0), minutes: 120, recurrence: 'biweekly' as const,
            sectionIds: [sectionIds.get('Foil')!],
        },
        {
            title: 'Vorstandssitzung', category: 'meeting' as const, location: 'Vereinsheim Büro',
            start: at(nextWeekday(3) + 7, 19, 0), minutes: 120, recurrence: 'monthly' as const,
            roleIds: [roleIds.get('Vorstand')!, roleIds.get('Kassenwart')!, roleIds.get('Schriftführer')!],
        },
        {
            title: 'Fehmarn Kite Cup', category: 'competition' as const, location: 'Fehmarn, Grüner Brink',
            start: at(nextWeekday(6) + 14, 0), allDayUntil: 1, recurrence: 'none' as const,
        },
        {
            title: 'Sommerfest am Strand', category: 'festival' as const, location: 'Vereinsheim',
            start: at(nextWeekday(5) + 21, 16, 0), minutes: 360, recurrence: 'none' as const,
            description: 'Grillen, Musik und Siegerehrung der Vereinsmeisterschaft. Familien willkommen!',
        },
    ];
    let youthTrainingId: string | undefined;
    for (const e of events) {
        const allDay = e.allDayUntil !== undefined;
        const end = allDay
            ? new Date(e.start.getFullYear(), e.start.getMonth(), e.start.getDate() + e.allDayUntil! + 1)
            : new Date(e.start.getTime() + e.minutes! * 60000);
        const created = await createEvent({
            title: e.title, category: e.category, location: e.location, description: e.description,
            start: e.start.toISOString(), end: end.toISOString(), allDay, timeZone: tz, recurrence: e.recurrence,
            groupIds: e.groupIds ?? [], sectionIds: e.sectionIds ?? [], roleIds: e.roleIds ?? [],
        });
        if (e.title === 'Jugendtraining Twintip') youthTrainingId = created.id;
    }

    // Attendance of the two past youth trainings: most were there, one was
    // excused and one missed a session.
    const youth = groupMembers.get('Jugend (U18)') ?? [];
    if (youthTrainingId && youth.length > 0) {
        const firstTraining = events[0].start;
        for (const [week, excused, missing] of [[0, 1, -1], [1, -1, 2]] as const) {
            await saveAttendance(youthTrainingId, ymd(addDays(firstTraining, 7 * week)), youth.map((memberId, i) => ({
                memberId,
                status: i === excused ? 'excused' : i === missing ? 'unexcused' : 'present',
            })));
        }
    }

    return {
        roles: roles.length,
        groups: groups.length,
        sections: sections.length,
        members: createdMembers.length,
        transactions: TRANSACTIONS.length,
        memberFees,
        inventoryItems: INVENTORY.length,
        inventoryLoans: loans.length,
    };
}

import { PrismaClient } from '@prisma/client';
import { faker } from '@faker-js/faker';

const prisma = new PrismaClient();

async function main() {
    console.log('⏳ Seeding data...');

    // Bestehende Daten löschen
    await prisma.member.deleteMany();
    await prisma.role.deleteMany();
    await prisma.group.deleteMany();
    await prisma.clubSection.deleteMany();

    // Initiale Daten einfügen
    await prisma.role.createMany({
        data: [{ name: 'Trainer' }, { name: 'Mitglied' }, { name: 'Admin' }]
    });

    await prisma.group.createMany({
        data: [{ name: 'Jugend' }, { name: 'Erwachsene' }, { name: 'Senioren' }]
    });

    await prisma.clubSection.createMany({
        data: [{ name: 'Fußball' }, { name: 'Tennis' }, { name: 'Schwimmen' }]
    });

    const allRoles = await prisma.role.findMany();
    const allGroups = await prisma.group.findMany();
    const allSections = await prisma.clubSection.findMany();

    // 100 Mitglieder anlegen
    for (let i = 0; i < 100; i++) {
        await prisma.member.create({
            data: {
                number: 1000 + i,
                firstName: faker.person.firstName(),
                lastName: faker.person.lastName(),
                email: faker.internet.email(),
                birthday: faker.date.birthdate({ min: 1950, max: 2010, mode: 'year' }).toISOString(),
                phone: faker.phone.number(),
                phoneMobile: faker.phone.number(),
                comment: faker.lorem.sentence(),
                entryDate: faker.date.past({ years: 5 }),
                exitDate: Math.random() < 0.1 ? faker.date.recent() : null,
                street: faker.location.streetAddress(),
                postalCode: faker.location.zipCode(),
                city: faker.location.city(),
                state: faker.location.state(),
                accountHolder: faker.person.fullName(),
                iban: faker.finance.iban(),
                bic: faker.finance.bic(),
                bankName: faker.company.name(),
                sepaMandateDate: faker.date.past(),
                roles: {
                    connect: [faker.helpers.arrayElement(allRoles)].map(r => ({ id: r.id }))
                },
                groups: {
                    connect: [faker.helpers.arrayElement(allGroups)].map(g => ({ id: g.id }))
                },
                sections: {
                    connect: [faker.helpers.arrayElement(allSections)].map(s => ({ id: s.id }))
                }
            }
        });
    }

    console.log('🌱 Seeding complete!');
}

main()
    .catch((e) => {
        console.error('❌ Error while seeding:', e);
        process.exit(1);
    })
    .finally(async () => {
        await prisma.$disconnect();
    });

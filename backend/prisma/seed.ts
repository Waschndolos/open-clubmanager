import {PrismaClient} from '@prisma/client';
import {faker} from '@faker-js/faker';
import * as bcrypt from "bcrypt";

const prisma = new PrismaClient({
        datasources: {
            db: {
                url: process.env.DATABASE_URL
            }
        }
    }
);
const nbMembers = 100;

async function main() {
    console.log('⏳ Seeding data...');

    // Delete current data
    await prisma.refreshToken.deleteMany();
    await prisma.userPreference.deleteMany();

    await prisma.user.deleteMany();

    await prisma.role.deleteMany();
    await prisma.group.deleteMany();
    await prisma.clubSection.deleteMany();
    await prisma.member.deleteMany();


    // Create initial data
    await prisma.role.createMany({
        data: [{name: 'Admin', id: 0}, {name: 'Trainer', id: 1}, {name: 'Member', id: 2}]
    });

    // Create groups and sections
    await prisma.group.createMany({
        data: [{name: 'Youth'}, {name: 'Adults'}, {name: 'Seniors'}]
    });

    await prisma.clubSection.createMany({
        data: [{name: 'Soccer'}, {name: 'Tennis'}, {name: 'Swim'}]
    });

    const allRoles = await prisma.role.findMany();
    const allGroups = await prisma.group.findMany();
    const allSections = await prisma.clubSection.findMany();

    const hashedPassword = await bcrypt.hash('admin', 10);
    // Create an initial user with an Admin role
    await prisma.user.create({
        data:
            {
                password: hashedPassword,
                email: 'admin@admin.com',
                roles: {
                    connect: {id: 0}
                }
            },

    })

    // Create members
    for (let i = 0; i < nbMembers; i++) {
        await prisma.member.create({
            data: {
                number: 1000 + i,
                firstName: faker.person.firstName(),
                lastName: faker.person.lastName(),
                email: faker.internet.email(),
                birthday: faker.date.birthdate({min: 1950, max: 2010, mode: 'year'}).toISOString(),
                phone: faker.phone.number(),
                phoneMobile: faker.phone.number(),
                comment: faker.lorem.sentence(),
                entryDate: faker.date.past({years: 5}),
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
                    connect: [faker.helpers.arrayElement(allRoles)].map(r => ({id: r.id}))
                },
                groups: {
                    connect: [faker.helpers.arrayElement(allGroups)].map(g => ({id: g.id}))
                },
                sections: {
                    connect: [faker.helpers.arrayElement(allSections)].map(s => ({id: s.id}))
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

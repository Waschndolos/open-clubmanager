import { PrismaClient } from '@prisma/client'

const prisma = new PrismaClient()

async function main() {
    console.log('🌱 Starting seed...')

    // Roles
    const roleAdmin = await prisma.role.upsert({
        where: { name: 'Admin' },
        update: {},
        create: { name: 'Admin' },
    })

    const roleMember = await prisma.role.upsert({
        where: { name: 'Member' },
        update: {},
        create: { name: 'Member' },
    })

    // Groups
    const groupYouth = await prisma.group.upsert({
        where: { name: 'Youth Group' },
        update: {},
        create: { name: 'Youth Group' },
    })

    // Sections
    const sectionA = await prisma.clubSection.upsert({
        where: { name: 'Section A' },
        update: {},
        create: { name: 'Section A' },
    })

    // Member
    await prisma.member.upsert({
        where: { number: 1 },
        update: {},
        create: {
            number: 1,
            entryDate: new Date('2024-01-01'),
            firstName: 'Max',
            lastName: 'Muster',
            email: 'max@example.com',
            iban: 'DE89370400440532013000',
            accountHolder: 'Max Muster',
            bankName: 'Musterbank',
            sepaMandateDate: new Date('2024-01-01'),
            roles: {
                connect: [{ id: roleMember.id }],
            },
            groups: {
                connect: [{ id: groupYouth.id }],
            },
            sections: {
                connect: [{ id: sectionA.id }],
            },
        },
    })

    console.log('✅ Seed data created successfully')
}

main()
    .then(() => prisma.$disconnect())
    .catch((e) => {
        console.error('❌ Error while seeding:', e)
        prisma.$disconnect().finally(() => process.exit(1))
    })

import { PrismaClient } from '@prisma/client'

const prisma = new PrismaClient()

async function main() {
    // Create roles
    const roleAdmin = await prisma.role.upsert({
        where: { name: 'Admin' },
        update: {},
        create: { name: 'Admin' }
    })

    const roleMember = await prisma.role.upsert({
        where: { name: 'Member' },
        update: {},
        create: { name: 'Member' }
    })

    // Create groups
    const groupYouth = await prisma.group.upsert({
        where: { name: 'Youth Group' },
        update: {},
        create: { name: 'Youth Group' }
    })

    // Create sections
    const sectionA = await prisma.clubSection.upsert({
        where: { name: 'Section A' },
        update: {},
        create: { name: 'Section A' }
    })

    // Create member
    await prisma.member.upsert({
        where: { number: 1 },
        update: {},
        create: {
            number: 1,
            firstName: 'Max',
            lastName: 'Muster',
            email: 'max@example.com',
            entryDate: '2024-01-01',
            iban: 'DE89370400440532013000',
            accountHolder: 'Max Muster',
            bankName: 'Musterbank',
            sepaMandateDate: '2024-01-01',
            roles: { connect: [{ id: roleMember.id }] },
            groups: { connect: [{ id: groupYouth.id }] },
            sections: { connect: [{ id: sectionA.id }] }
        }
    })
}

main()
    .then(() => {
        console.log('✅ Seed data created')
        return prisma.$disconnect()
    })
    .catch((e) => {
        console.error('❌ Error seeding data:', e)
        return prisma.$disconnect().finally(() => process.exit(1))
    })

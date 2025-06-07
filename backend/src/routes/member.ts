import express from 'express'
import { PrismaClient } from '@prisma/client'
import dotenv from 'dotenv'
dotenv.config()
const prisma = new PrismaClient({
    datasources: {
        db: {
            url: process.env.DATABASE_URL
        }
    }
})
const router = express.Router()

// GET /api/members – List all members
router.get('/', async (_req, res) => {
    try {
        const members = await prisma.member.findMany({
            include: {
                roles: true,
                groups: true,
                sections: true
            }
        })
        res.json(members)
    } catch (err) {
        console.error(err)
        res.status(500).json({ error: 'Error loading members' })
    }
})

// POST /api/members – Create new member
router.post('/', async (req, res) => {
    try {
        const data = req.body

        const newMember = await prisma.member.create({
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
                roles: {
                    connect: data.roleIds?.map((id: number) => ({ id })) || []
                },
                groups: {
                    connect: data.groupIds?.map((id: number) => ({ id })) || []
                },
                sections: {
                    connect: data.sectionIds?.map((id: number) => ({ id })) || []
                }
            }
        })

        res.json(newMember)
    } catch (err) {
        console.error(err)
        res.status(500).json({ error: 'Error creating member' })
    }
})

// update Member
router.put('/:id', async (req, res) => {
    try {
        const id = parseInt(req.params.id)
        const data = req.body

        const updated = await prisma.member.update({
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
                roles: {
                    set: data.roleIds?.map((id: number) => ({ id })) || []
                },
                groups: {
                    set: data.groupIds?.map((id: number) => ({ id })) || []
                },
                sections: {
                    set: data.sectionIds?.map((id: number) => ({ id })) || []
                }
            }
        })

        res.json(updated)
    } catch (err) {
        console.error(err)
        res.status(500).json({ error: 'Error updating members' })
    }
})

// delete members
router.delete('/:id', async (req, res) => {
    try {
        const id = parseInt(req.params.id)

        await prisma.member.delete({
            where: { id }
        })

        res.json({ success: true })
    } catch (err) {
        console.error(err)
        res.status(500).json({ error: 'Error deleting members' })
    }
})


export default router

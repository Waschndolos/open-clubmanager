import { Router } from 'express';
import { getClient } from '../../../db.ts';
import { asyncHandler } from '../../core/asyncHandler.ts';

interface Member {
    firstName: string;
    lastName: string;
    birthday: Date | null;
    exitDate: Date | null;
}

function formatDate(date: Date | null): string {
    if (!date) return 'unknown';
    return date.toISOString().slice(0, 10);
}

function isBirthdayWithinNextTwoWeeks(birthday: Date | null): boolean {
    if (!birthday) return false;

    const today = new Date();
    const thisYear = today.getFullYear();
    const birthdayThisYear = new Date(thisYear, birthday.getMonth(), birthday.getDate());

    if (birthdayThisYear < today) {
        birthdayThisYear.setFullYear(thisYear + 1);
    }

    const twoWeeksLater = new Date(today);
    twoWeeksLater.setDate(today.getDate() + 14);

    return birthdayThisYear >= today && birthdayThisYear <= twoWeeksLater;
}

function isMemberExitingThisYear(exitDate: Date | null): boolean {
    if (!exitDate) return false;
    const thisYear = new Date().getFullYear();
    return exitDate.getFullYear() === thisYear;
}

export function createStatisticsRoutes(): Router {
    const router = Router();

    router.get('/', asyncHandler(async (_req, res) => {
        const prisma = await getClient();
        const members: Member[] = await prisma.member.findMany();

        const upcoming = members.filter((member) => isBirthdayWithinNextTwoWeeks(member.birthday));
        const exiting = members.filter((member) => isMemberExitingThisYear(member.exitDate));

        res.json([
            { id: 1, value: members.length },
            {
                id: 2,
                value: upcoming.length,
                details: upcoming
                    .sort((a, b) => {
                        const today = new Date();
                        const thisYear = today.getFullYear();
                        const aDate = new Date(thisYear, a.birthday!.getMonth(), a.birthday!.getDate());
                        const bDate = new Date(thisYear, b.birthday!.getMonth(), b.birthday!.getDate());
                        return aDate.getTime() - bDate.getTime();
                    })
                    .map((member) => `${member.firstName} ${member.lastName} (${formatDate(member.birthday)})`),
            },
            {
                id: 3,
                value: exiting.length,
                details: exiting.map((member) => `${member.firstName} ${member.lastName} (${formatDate(member.exitDate)})`),
            },
        ]);
    }));

    return router;
}


import express from 'express';
import { getClient } from "../db.ts";

const router = express.Router();

interface Member {
    firstName: string;
    lastName: string;
    birthday: Date | null;
    exitDate: Date | null;
}

router.get('/', async (req, res) => {
    try {
        const prisma = await getClient();
        const members: Member[] = await prisma.member.findMany();

        const statistics = [
            buildMemberCountStat(members),
            buildUpcomingBirthdaysStat(members),
            buildExitingMembersStat(members)
        ];

        res.json(statistics);
    } catch (err) {
        console.error(err);
        res.status(500).json({ error: "Failed to fetch statistics" });
    }
});

function buildMemberCountStat(members: Member[]) {
    return {
        id: 1,
        value: members.length,
    };
}

function buildUpcomingBirthdaysStat(members: Member[]) {
    const upcoming = members.filter(m => isBirthdayWithinNextTwoWeeks(m.birthday));

    return {
        id: 2,
        value: upcoming.length,
        details: upcoming.map(m => `${m.firstName} ${m.lastName} (${formatDate(m.birthday)})`),
    };
}

function buildExitingMembersStat(members: Member[]) {
    const exiting = members.filter(m => isMemberExitingThisYear(m.exitDate));

    return {
        id: 3,
        value: exiting.length,
        details: exiting.map(m => `${m.firstName} ${m.lastName} (${formatDate(m.exitDate)})`),
    };
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

function formatDate(date: Date | null): string {
    if (!date) return "unknown";
    return date.toISOString().slice(0, 10); // YYYY-MM-DD
}

export default router;

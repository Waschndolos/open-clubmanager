import {isAxiosError} from "axios";
import {Member} from "../../api/types";

/**
 * Checks of the member dialog. They mirror the server's rules
 * (server/internal/store/membervalidation.go), which answers invalid members
 * with 400 VALIDATION_FAILED and the same field/code pairs.
 */
export type FieldProblem = { code: string; limit?: number };
export type FieldProblems = Partial<Record<keyof Member, FieldProblem>>;

const limits: Partial<Record<keyof Member, number>> = {
    firstName: 100, lastName: 100, email: 254, comment: 2000,
    phone: 50, phoneMobile: 50, postalCode: 50,
    street: 200, city: 200, state: 200, accountHolder: 200, bankName: 200,
};

// IBAN length per country (SWIFT IBAN registry), as on the server.
const ibanLengths: Record<string, number> = {
    AD: 24, AE: 23, AL: 28, AT: 20, AZ: 28, BA: 20, BE: 16, BG: 22, BH: 22, BI: 27,
    BR: 29, BY: 28, CH: 21, CR: 22, CY: 28, CZ: 24, DE: 22, DJ: 27, DK: 18, DO: 28,
    EE: 20, EG: 29, ES: 24, FI: 18, FK: 18, FO: 18, FR: 27, GB: 22, GE: 22, GI: 23,
    GL: 18, GR: 27, GT: 28, HR: 21, HU: 28, IE: 22, IL: 23, IQ: 23, IS: 26, IT: 27,
    JO: 30, KW: 30, KZ: 20, LB: 28, LC: 32, LI: 21, LT: 20, LU: 20, LV: 21, LY: 25,
    MC: 27, MD: 24, ME: 22, MK: 19, MN: 20, MR: 27, MT: 31, MU: 30, NI: 28, NL: 18,
    NO: 15, OM: 23, PK: 24, PL: 28, PS: 29, PT: 25, QA: 29, RO: 24, RS: 22, RU: 33,
    SA: 24, SC: 31, SD: 18, SE: 24, SI: 19, SK: 24, SM: 27, SO: 23, ST: 25, SV: 28,
    TL: 23, TN: 24, TR: 26, UA: 29, VA: 22, VG: 24, XK: 20, YE: 30,
};

/** Removes spaces and upper-cases an IBAN or BIC as typed by a person. */
export const normalizeIban = (value: string) => value.replace(/\s+/g, "").toUpperCase();

/** 'invalid' (characters, country, length), 'checksum' (mod 97) or undefined when valid. */
export function ibanProblem(value: string): "invalid" | "checksum" | undefined {
    const iban = normalizeIban(value);
    if (!/^[A-Z]{2}[0-9]{2}[A-Z0-9]{11,30}$/.test(iban) || ibanLengths[iban.slice(0, 2)] !== iban.length) {
        return "invalid";
    }
    // ISO 7064 MOD 97-10, digit by digit so no big integers are needed.
    let rest = 0;
    for (const ch of iban.slice(4) + iban.slice(0, 4)) {
        const digits = /[0-9]/.test(ch) ? ch : String(ch.charCodeAt(0) - 55);
        for (const d of digits) rest = (rest * 10 + Number(d)) % 97;
    }
    return rest === 1 ? undefined : "checksum";
}

export const validBic = (value: string) => /^[A-Z]{6}[A-Z2-9][A-NP-Z0-9]([A-Z0-9]{3})?$/.test(normalizeIban(value));

export const validEmail = (value: string) => /^[^\s@<>()]+@[^\s@<>()]+$/.test(value.trim());

/** The calendar day of a date field (sent as UTC midnight, see EditMemberDialog). */
const calendarDay = (value?: string | Date) =>
    value ? (typeof value === "string" ? value : value.toISOString()).slice(0, 10) : undefined;

const localToday = () => {
    const now = new Date();
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
};

/**
 * Checks a member like the server. Only fields in `fields` are checked, so a
 * role without access to bank or contact data does not see problems it cannot fix.
 */
export function validateMember(member: Member, fields: { core: boolean; contact: boolean; bank: boolean }): FieldProblems {
    const problems: FieldProblems = {};
    const text = (key: keyof Member) => {
        const value = member[key];
        const limit = limits[key];
        if (typeof value === "string" && limit && [...value.trim()].length > limit) {
            problems[key] = {code: "too_long", limit};
        }
    };

    if (fields.core) {
        if (!Number.isInteger(member.number) || (member.number ?? 0) < 1) problems.number = {code: "not_positive"};
        for (const key of ["firstName", "lastName"] as const) {
            if (!member[key]?.trim()) problems[key] = {code: "required"};
            else text(key);
        }
        const birthday = calendarDay(member.birthday);
        if (birthday && birthday > localToday()) problems.birthday = {code: "in_future"};
        const entry = calendarDay(member.entryDate);
        const exit = calendarDay(member.exitDate);
        if (entry && exit && exit < entry) problems.exitDate = {code: "before_entry"};
        text("comment");
    }
    if (fields.contact) {
        const email = member.email?.trim();
        if (!email) problems.email = {code: "required"};
        else if (!validEmail(email)) problems.email = {code: "invalid"};
        else text("email");
        (["phone", "phoneMobile", "street", "postalCode", "city", "state"] as const).forEach(text);
    }
    if (fields.bank) {
        const iban = member.iban?.trim();
        const problem = iban ? ibanProblem(iban) : undefined;
        if (problem) problems.iban = {code: problem};
        if (member.bic?.trim() && !validBic(member.bic)) problems.bic = {code: "invalid"};
        text("accountHolder");
        text("bankName");
    }
    return problems;
}

type Translate = (key: string | string[], options?: Record<string, unknown>) => string;

/** The message for a field problem (members.validation.*). */
export const problemText = (t: Translate, field: string, problem: FieldProblem) =>
    t([`members.validation.${field}_${problem.code}`, `members.validation.${problem.code}`, "members.validation.invalid"],
        {limit: problem.limit});

/** The field problems of a 400 VALIDATION_FAILED answer, or undefined for other errors. */
export function serverFieldProblems(e: unknown): FieldProblems | undefined {
    if (!isAxiosError(e) || e.response?.status !== 400 || e.response.data?.code !== "VALIDATION_FAILED") return undefined;
    const problems: FieldProblems = {};
    for (const f of (e.response.data.fields ?? []) as { field: string; code: string; limit?: number }[]) {
        problems[f.field as keyof Member] = {code: f.code, limit: f.limit};
    }
    return problems;
}

import { XMLParser } from 'fast-xml-parser';
import { HttpError } from '../../core/httpError.ts';

export interface ParsedBankTransaction {
    date: Date;
    description: string;
    amount: number;
    type: 'income' | 'expense';
    category: string;
    notes: string;
}

type UnknownRecord = Record<string, unknown>;

const parser = new XMLParser({
    ignoreAttributes: false,
    attributeNamePrefix: '@_',
    parseTagValue: false,
    trimValues: true,
});

function asArray<T>(value: T | T[] | undefined): T[] {
    if (value === undefined) {
        return [];
    }
    return Array.isArray(value) ? value : [value];
}

function getText(value: unknown): string | undefined {
    if (value === null || value === undefined) {
        return undefined;
    }

    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
        const text = String(value).trim();
        return text.length > 0 ? text : undefined;
    }

    if (Array.isArray(value)) {
        for (const item of value) {
            const candidate = getText(item);
            if (candidate) {
                return candidate;
            }
        }
        return undefined;
    }

    if (typeof value === 'object') {
        const record = value as UnknownRecord;
        for (const key of ['#text', 'DtTm', 'Dt', 'Ustrd', 'AddtlNtryInf']) {
            const candidate = getText(record[key]);
            if (candidate) {
                return candidate;
            }
        }
    }

    return undefined;
}

function parseAmount(entry: UnknownRecord): number {
    const amountRaw = entry.Amt;
    const amountText = getText(amountRaw);
    const amount = Number(amountText);

    if (!Number.isFinite(amount)) {
        throw new HttpError(400, 'FINANCE_IMPORT_INVALID_AMOUNT', 'CAMT.053 entry contains an invalid amount.');
    }

    return Math.abs(amount);
}

function parseDate(entry: UnknownRecord): Date {
    const bookingDate = getText((entry.BookgDt as UnknownRecord | undefined)?.DtTm)
        ?? getText((entry.BookgDt as UnknownRecord | undefined)?.Dt)
        ?? getText((entry.ValDt as UnknownRecord | undefined)?.DtTm)
        ?? getText((entry.ValDt as UnknownRecord | undefined)?.Dt);

    if (!bookingDate) {
        throw new HttpError(400, 'FINANCE_IMPORT_INVALID_DATE', 'CAMT.053 entry contains no booking date.');
    }

    const parsed = new Date(bookingDate);
    if (Number.isNaN(parsed.getTime())) {
        throw new HttpError(400, 'FINANCE_IMPORT_INVALID_DATE', 'CAMT.053 entry contains an invalid booking date.');
    }

    return parsed;
}

function parseType(entry: UnknownRecord): 'income' | 'expense' {
    const indicator = getText(entry.CdtDbtInd)?.toUpperCase();
    if (indicator === 'CRDT') {
        return 'income';
    }
    if (indicator === 'DBIT') {
        return 'expense';
    }

    throw new HttpError(400, 'FINANCE_IMPORT_INVALID_TYPE', 'CAMT.053 entry contains an unknown CdtDbtInd value.');
}

function parseDescription(entry: UnknownRecord): string {
    const remittance = getText(
        (((entry.NtryDtls as UnknownRecord | undefined)?.TxDtls as UnknownRecord | undefined)?.RmtInf as UnknownRecord | undefined)?.Ustrd
    );

    const directRemittance = getText((entry.RmtInf as UnknownRecord | undefined)?.Ustrd);
    const additionalInfo = getText(entry.AddtlNtryInf);

    return remittance
        ?? directRemittance
        ?? additionalInfo
        ?? 'Bank transaction import';
}

export function parseCamt053(xml: string): ParsedBankTransaction[] {
    if (!xml.trim()) {
        throw new HttpError(400, 'FINANCE_IMPORT_EMPTY_FILE', 'No CAMT.053 XML content provided.');
    }

    let parsedXml: UnknownRecord;
    try {
        parsedXml = parser.parse(xml) as UnknownRecord;
    } catch {
        throw new HttpError(400, 'FINANCE_IMPORT_INVALID_XML', 'Invalid CAMT.053 XML document.');
    }

    const documentNode = parsedXml.Document as UnknownRecord | undefined;
    const customerStatement = documentNode?.BkToCstmrStmt as UnknownRecord | undefined;
    const statementNode = customerStatement?.Stmt as UnknownRecord | undefined;
    const entries = asArray(statementNode?.Ntry);

    if (entries.length === 0) {
        throw new HttpError(400, 'FINANCE_IMPORT_NO_ENTRIES', 'No booking entries found in CAMT.053 document.');
    }

    return entries.map((entry) => {
        const entryRecord = (entry ?? {}) as UnknownRecord;
        return {
            date: parseDate(entryRecord),
            description: parseDescription(entryRecord),
            amount: parseAmount(entryRecord),
            type: parseType(entryRecord),
            category: 'bank-import',
            notes: 'Imported from CAMT.053',
        };
    });
}


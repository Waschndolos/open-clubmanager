import { XMLParser } from 'fast-xml-parser';
import { HttpError } from '../../core/httpError.ts';

export interface ParsedBankTransaction {
    date: Date;
    description: string;
    amount: number;
    type: 'income' | 'expense';
    category: string;
    notes: string;
    counterpartyName: string | null;
    counterpartyIban: string | null;
    endToEndId: string | null;
    mandateId: string | null;
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

function getFirstTxDetails(entry: UnknownRecord): UnknownRecord | undefined {
    const details = entry.NtryDtls as UnknownRecord | UnknownRecord[] | undefined;
    const detail = asArray(details)[0] as UnknownRecord | undefined;
    const txDetails = detail?.TxDtls as UnknownRecord | UnknownRecord[] | undefined;
    return asArray(txDetails)[0] as UnknownRecord | undefined;
}

function parseCounterpartyName(entry: UnknownRecord): string | null {
    const tx = getFirstTxDetails(entry);
    const related = tx?.RltdPties as UnknownRecord | undefined;
    const debtorName = getText((related?.Dbtr as UnknownRecord | undefined)?.Nm);
    const creditorName = getText((related?.Cdtr as UnknownRecord | undefined)?.Nm);
    return debtorName ?? creditorName ?? null;
}

function parseCounterpartyIban(entry: UnknownRecord): string | null {
    const tx = getFirstTxDetails(entry);
    const related = tx?.RltdPties as UnknownRecord | undefined;
    const debtorIban = getText((((related?.DbtrAcct as UnknownRecord | undefined)?.Id as UnknownRecord | undefined)?.IBAN));
    const creditorIban = getText((((related?.CdtrAcct as UnknownRecord | undefined)?.Id as UnknownRecord | undefined)?.IBAN));
    return debtorIban ?? creditorIban ?? null;
}

function parseEndToEndId(entry: UnknownRecord): string | null {
    const tx = getFirstTxDetails(entry);
    const refs = tx?.Refs as UnknownRecord | undefined;
    return getText(refs?.EndToEndId) ?? null;
}

function parseMandateId(entry: UnknownRecord): string | null {
    const tx = getFirstTxDetails(entry);
    const related = tx?.RltdTxInf as UnknownRecord | undefined;
    const mandate = related?.MndtRltdInf as UnknownRecord | undefined;
    return getText(mandate?.MndtId) ?? null;
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
        const description = parseDescription(entryRecord);
        const counterpartyName = parseCounterpartyName(entryRecord);
        const counterpartyIban = parseCounterpartyIban(entryRecord);
        const endToEndId = parseEndToEndId(entryRecord);
        const mandateId = parseMandateId(entryRecord);

        const notesParts = [
            'Imported from CAMT.053',
            counterpartyName ? `Counterparty: ${counterpartyName}` : null,
            counterpartyIban ? `IBAN: ${counterpartyIban}` : null,
            endToEndId ? `EndToEndId: ${endToEndId}` : null,
            mandateId ? `MandateId: ${mandateId}` : null,
        ].filter((item): item is string => Boolean(item));

        return {
            date: parseDate(entryRecord),
            description,
            amount: parseAmount(entryRecord),
            type: parseType(entryRecord),
            category: 'bank-import',
            notes: notesParts.join(' | '),
            counterpartyName,
            counterpartyIban,
            endToEndId,
            mandateId,
        };
    });
}


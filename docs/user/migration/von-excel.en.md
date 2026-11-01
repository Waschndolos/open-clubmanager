# Switching from Excel

Most clubs start with a member list in a spreadsheet. The import brings it in within minutes.

## 1. Preparing the Excel list

Open ClubManager reads `.xlsx` files. The **first row** must contain the column headers, followed by one member per row.

| Field in Open ClubManager | Typical column in your list |
|---|---|
| Membership number (`number`) | Member no. |
| First name (`firstName`), last name (`lastName`) | First name, Surname |
| Email (`email`) – **required** | Email |
| Phone (`phone`), mobile (`phoneMobile`) | Phone, Mobile |
| Birthday (`birthday`) | Date of birth |
| Street (`street`), postal code (`postalCode`), city (`city`), state (`state`) | Address |
| Entry date (`entryDate`), exit date (`exitDate`) | Joined, Left |
| Account holder (`accountHolder`), IBAN (`iban`), BIC (`bic`), bank (`bankName`) | Bank details |
| SEPA mandate date (`sepaMandateDate`) | Mandate dated |
| Comment (`comment`) | Notes |

Tips:

- **One column per field.** If "street, postal code city" is in one cell, split it first in your spreadsheet (Data → Text to columns).
- Format **dates** as real spreadsheet dates, not as text.
- **Email is required.** Members without an email address get a unique placeholder (e.g. `member-1042@club.invalid`) that you replace later.
- **Membership numbers and email addresses must be unique.** Duplicates are highlighted in the preview.
- If your columns are named exactly like the field names in brackets, Open ClubManager maps them automatically.

## 2. Running the import

1. **Members → import icon** (:material-upload:) → **Select Excel-File**.
2. **Assign columns:** choose the matching field for each column of your file, or "Ignore".
3. **Update existing members by:** email or membership number. For the first import it doesn't matter; for a later re-import, existing members are updated instead of duplicated.
4. Check the **preview** and import.

Rows with invalid details (e.g. a missing name or an IBAN with wrong check digits, see [input checks](../funktionen/mitglieder.md#adding-or-editing-a-member)) are skipped; the other rows are imported. A notification then lists which rows are missing and why. Fix them in the file and import again.

## 3. Follow-up

- Create **sections, groups and roles** in the [club structure](../funktionen/vereinsstruktur.md) and assign them in bulk via **Assign members**.
- Create **households** for family memberships ([Families & households](../funktionen/haushalte.md)).
- Record the current year's **fees** under [Finances → Member Fees](../funktionen/finanzen.md#member-fees).
- **Take samples**, then archive or securely delete the old spreadsheet.

## Other lists

| Spreadsheet | Where it goes |
|---|---|
| Cash book | Start fresh from a cut-off date; record the opening balance as one income transaction; bank movements via [CAMT.053 import](../funktionen/finanzen.md#importing-a-bank-statement-camt053) |
| Inventory list | enter in [Inventory](../funktionen/inventar.md) (no import yet) |
| Schedule | in [Events](../funktionen/termine.md); create training sessions as a series |
| Licences / first-aid list | in [Qualifications](../funktionen/qualifikationen.md) |

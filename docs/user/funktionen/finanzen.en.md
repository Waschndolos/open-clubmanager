---
routes:
  - /finance
---

# Finances

Cash book, member fees and fee collection by SEPA direct debit. Only visible to **administrators** and **treasurers**.

![Cash book](../assets/screenshots/finance.png){ .screenshot }

At the top you see the **balance**, **total income**, **total expenses** and **outstanding fees**.

## Cash book

Every income and expense is a transaction with date, description, amount, type (income/expense), category and notes.

- **Add Transaction** creates a new transaction.
- Use the pencil and bin icons to edit or delete transactions.

Costs from [maintenance in the inventory](inventar.md#inspection-maintenance) can be booked straight into the cash book as an expense.

### Importing a bank statement (CAMT.053)

Instead of typing every transaction, import your bank's statement in **CAMT.053** format (XML). Most banks offer this in online banking, often under "export statement" or "export transactions → CAMT".

1. Click **Import CAMT.053** and select the file.
2. Open ClubManager creates a transaction for every bank movement: credits as income, debits as expenses.
3. Afterwards you see a summary of how many transactions were imported.

!!! warning "Current limitations"
    - Import each statement **only once**. Already imported movements are not detected yet and would be booked twice.
    - Incoming payments are **not yet matched to members automatically**. Mark paid fees as paid under **Member Fees** yourself.

## Member fees

![Member fees](../assets/screenshots/finance-fees.png){ .screenshot }

Here you manage the fees per member and year: amount, due date, paid date and description. Open fees are marked as **Unpaid**.

## Direct debit (SEPA)

![Direct debit runs](../assets/screenshots/finance-direct-debit.png){ .screenshot }

In the **Direct debit** tab you collect open fees by SEPA Core direct debit. Open ClubManager creates a file in the **pain.008** bank format that you upload in online banking. The tab only appears if your role may edit finances and, with [collaboration](../zusammenarbeit/index.md), your device receives bank and finance data.

### One-time setup

1. In the [settings](einstellungen.md#sepa-direct-debit) under **SEPA direct debit**, enter the club name, the **creditor identifier** (issued by your national bank, e.g. `DE98ZZZ09999999999`), the **IBAN of the club account** and optionally a **remittance template**.
2. For every member who pays by direct debit, enter the IBAN, **SEPA Mandate Date** and a unique **mandate reference** (e.g. `NW-1001`) under **Bank Details** in the [member dialog](mitglieder.md#adding-or-editing-a-member).

### Creating a direct debit run

Click **Create direct debit run**. A wizard guides you through three steps:

1. **Select fees:** all open fees are listed. Fees that can't be collected (no IBAN, no mandate, no mandate reference, invalid IBAN …) are listed under **Cannot be collected** with the reason; complete the member's data.
2. **Due date & remittance:** the **collection date**. The file must reach the bank at least one business day before. Below it you see a preview of the remittance text.
3. **Review & export:** check the count, total, first (FRST) and recurring (RCUR) collections, then click **Create & download**.

The fees are then marked as **Being collected**.

### After the export

| Step | Where |
|---|---|
| Upload the `.xml` file in online banking and authorise it | at your bank |
| Once the bank has collected: **Confirm as paid** and enter the date; all fees of the run are marked as paid | the run in the list |
| Record returned debits on the individual fee | [Member fees](#member-fees) |
| Created a run by mistake? **Cancel run** and the fees are open again. Then don't upload the file, or revoke the order at the bank | the run in the list |

**Download XML file** gives you a run's file again.

!!! info "First and recurring collections"
    When a mandate is used for the first time, Open ClubManager marks the debit as a first collection (FRST), afterwards as recurring (RCUR). This happens automatically.

## Deleting the full ledger

For a fresh start (e.g. after trying things out with demo data) you can delete all transactions and fees at once. For safety you have to type a confirmation word. **This cannot be undone**; create a [backup](einstellungen.md#backup) first.

!!! warning "Not a replacement for accounting"
    The cash book is a helper for everyday club life. It does not replace tax-compliant bookkeeping, accounting exports or advice from tax advisors or auditors.

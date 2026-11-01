---
routes:
  - /finance
---

# Finances

Cash book and member fees. Only visible to **administrators** and **treasurers**.

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

## Deleting the full ledger

For a fresh start (e.g. after trying things out with demo data) you can delete all transactions and fees at once. For safety you have to type a confirmation word. **This cannot be undone**; create a [backup](einstellungen.md#backup) first.

!!! warning "Not a replacement for accounting"
    The cash book is a helper for everyday club life. It does not replace tax-compliant bookkeeping, accounting exports or advice from tax advisors or auditors.

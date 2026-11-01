---
routes:
  - /finance
---

# Finances

Cash book, member fees, fee collection by SEPA direct debit and payment reconciliation with the bank statement. Only visible to **administrators** and **treasurers**.

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
2. Open ClubManager creates a transaction for every bank movement: credits as income, debits as expenses. The payer's name and IBAN and the direct debit references are kept for the [payment reconciliation](#payment-reconciliation).
3. Afterwards you see a summary of how many transactions were imported, and for how many there is a suggestion which open fee they pay. **Open reconciliation** takes you straight to the review.

!!! warning "Current limitations"
    Import each statement **only once**. Already imported movements are not detected yet and would be booked twice.

## Member fees

![Member fees](../assets/screenshots/finance-fees.png){ .screenshot }

Here you manage the fees per member and year: amount, due date, paid date and description. Open fees are marked as **Unpaid**.

A fee counts as paid once you enter the paid date, a [direct debit run](#direct-debit-sepa) is confirmed, or enough payments are assigned to it in the [payment reconciliation](#payment-reconciliation). A **partly paid** fee stays unpaid until the rest arrives.

## Direct debit (SEPA)

![Direct debit runs](../assets/screenshots/finance-direct-debit.png){ .screenshot }

In the **Direct debit** tab you collect open fees by SEPA Core direct debit. Open ClubManager creates a file in the **pain.008** bank format that you upload in online banking. The tab only appears if your role may edit finances and, with [collaboration](../zusammenarbeit/index.md), your device receives bank and finance data.

### One-time setup

1. In the [settings](einstellungen.md#sepa-direct-debit) under **SEPA direct debit**, enter the club name, the **creditor identifier** (issued by your national bank, e.g. `DE98ZZZ09999999999`), the **IBAN of the club account** and optionally a **remittance template**.
2. For every member who pays by direct debit, enter the IBAN, **SEPA Mandate Date** and a unique **mandate reference** (e.g. `NW-1001`) under **Bank Details** in the [member dialog](mitglieder.md#adding-or-editing-a-member).

### Creating a direct debit run

Click **Create direct debit run**. A wizard guides you through three steps:

1. **Select fees:** all open fees are listed; partly paid fees with the amount still due. Fees that can't be collected (no IBAN, no mandate, no mandate reference, invalid IBAN …) are listed under **Cannot be collected** with the reason; complete the member's data.
2. **Due date & remittance:** the **collection date**. The file must reach the bank at least one business day before. Below it you see a preview of the remittance text.
3. **Review & export:** check the count, total, first (FRST) and recurring (RCUR) collections, then click **Create & download**.

The fees are then marked as **Being collected**.

### After the export

| Step | Where |
|---|---|
| Upload the `.xml` file in online banking and authorise it | at your bank |
| Once the bank has collected: **Confirm as paid** and enter the date; all fees of the run are marked as paid | the run in the list |
| Returned debits: import the bank statement and confirm them in the reconciliation; the fee is open again and can be collected once more | [Payment reconciliation](#returned-direct-debits) |
| Created a run by mistake? **Cancel run** and the fees are open again. Then don't upload the file, or revoke the order at the bank | the run in the list |

**Download XML file** gives you a run's file again.

!!! info "First and recurring collections"
    When a mandate is used for the first time, Open ClubManager marks the debit as a first collection (FRST), afterwards as recurring (RCUR). This happens automatically.

## Payment reconciliation

![Payment reconciliation](../assets/screenshots/finance-reconciliation.png){ .screenshot }

In the **Reconciliation** tab you assign imported incoming payments to open fees instead of looking up every transfer by hand. Open ClubManager suggests which fees each transaction pays; **nothing is booked until you confirm**. Like [direct debit](#direct-debit-sepa), the tab only appears if your role may edit finances and your device receives bank and finance data.

### How suggestions are made

For every incoming payment Open ClubManager checks:

- the **member number** in the remittance text, e.g. "Fee 2026 member 1013" or "No. 12" (short numbers only with a word such as "Mitglied", "member", "Nr.", "No." or "#" in front),
- the **payer's IBAN** against the members' bank details,
- the **name** of the payer and in the remittance text, also with umlauts ("Möller" = "Moeller") and small typos,
- the **amount**: does it match one open fee, all open fees of the member, or of their [household](haushalte.md)?

This gives the **match quality**:

| Match quality | Meaning |
|---|---|
| **Certain** | Member number or IBAN recognised and the amount matches exactly |
| **Probable** | Member number or IBAN recognised but the amount differs (partial or overpayment), or name and amount match |
| **Unclear** | Only the name matches, not the amount; check carefully |
| **No suggestion** | No matching open fee, e.g. for donations or sponsoring |

The chips below a suggestion show what it was recognised by (e.g. *Member number*, *IBAN*, *Amount matches*, *Household*, *Partial payment*).

### Reviewing and confirming

- **Confirm** (tick) takes over the suggestion. The fee gets the booking date as paid date.
- **Confirm all certain** takes over all suggestions rated *Certain* at once.
- **Change assignment** (pencil) opens a searchable list of all open fees. Tick the right fees and adjust the amounts; this is how you correct a suggestion or assign a transaction without one yourself.
- **Mark as reviewed without fee** is for a transaction that pays no fee (e.g. a donation). It disappears from the open transactions.
- **Undo** (under **Done**) takes an assignment back. The fees return to their previous state.

### Partial payments and collective transfers

- If a member pays in **instalments**, assign every instalment to the same fee. The fee stays *Unpaid* until the total is reached; a direct debit then only collects the rest.
- If parents pay for **several children** at once, the suggestion splits the amount across the household's fees. In the dialog you can pick any number of fees, also of different members, each with its own amount.
- If an overpayment leaves a rest, it is shown and not assigned to any fee.

### Returned direct debits

Open ClubManager recognises a returned direct debit on import by its bank code or return reason and shows it in red with the reason, e.g. *Insufficient funds* or *Refund requested by payer*. The suggestion names the fee that had been paid: reliably by the debit reference for runs created in Open ClubManager, otherwise by IBAN, member number and amount.

When you confirm the returned debit, the fee is **open** again and can be collected in a new [direct debit run](#direct-debit-sepa). Bank charges for the return remain as a rest without fee.

!!! tip "Trying it with the demo data"
    If you chose the demo data during setup, you can try the reconciliation with the example file [`testdata/camt053-reconciliation.xml`](https://github.com/Waschndolos/open-clubmanager/blob/main/testdata/camt053-reconciliation.xml): it contains certain, probable and unclear payments, a family transfer, a returned direct debit and a sponsoring income.

## Deleting the full ledger

For a fresh start (e.g. after trying things out with demo data) you can delete all transactions and fees at once. For safety you have to type a confirmation word. **This cannot be undone**; create a [backup](einstellungen.md#backup) first.

!!! warning "Not a replacement for accounting"
    The cash book is a helper for everyday club life. It does not replace tax-compliant bookkeeping, accounting exports or advice from tax advisors or auditors.

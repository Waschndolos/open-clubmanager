---
routes:
  - /history
---

# History

The history shows who changed what and when.

![History](../assets/screenshots/history.png){ .screenshot }

Each entry contains:

| Column | Content |
|---|---|
| **Timestamp** | when the change was made |
| **Action** | CREATE, UPDATE or DELETE |
| **Entity** | what was changed, e.g. Member, Event, InventoryLoan |
| **ID** | the internal identifier of the entry |
| **User** | who made the change |
| **Details** | further information about the change |

The history helps with questions ("Who changed the address?") and as evidence for auditors. It cannot be edited.

Entries about finances – transactions, fees and fee models, direct debits, payment reconciliation and donation receipts – are only shown to roles that may see the finances (Administrator and Treasurer). See [Users & roles](../erste-schritte/rollen.md).

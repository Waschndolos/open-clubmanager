---
routes:
  - /collaboration/conflicts
---

# Needs attention

Because every device also works offline, two people can assign the same unique value at the same time, for example both giving membership number 1042 to two different new members. Open ClubManager detects this and shows it under **Collaboration → To resolve** (also via the sync status "Please have a look").

## What can be duplicated?

- **Membership numbers**
- **Email addresses** of members
- **Names** of groups, roles and sections

## How to resolve it

For each case, Open ClubManager shows which entry **keeps the value** and which one **must be changed**. The rule: whoever assigned the value **first** keeps it.

| Case | Resolution |
|---|---|
| Duplicate membership number | **Use the next free number** or **Choose a number** |
| Duplicate email address | enter a new email address |
| Duplicate name | enter a new name |

Click **Apply**. The change is distributed to everyone and the entry disappears from the list.

Anyone with write permission for the affected area can resolve it. Those who may only read see the notice but can't change anything.

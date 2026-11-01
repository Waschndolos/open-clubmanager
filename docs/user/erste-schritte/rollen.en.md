# Users & roles

Open ClubManager has two kinds of permissions that work together:

1. **User role**: which account you log in with and what this account may do. Always applies.
2. **Device role**: only with [collaboration](../zusammenarbeit/index.md) turned on: which data areas this device receives at all.

The stricter of the two always wins.

## User roles

| Role | May |
|---|---|
| **Administrator** | do everything, including user management |
| **Treasurer** | edit club data, see and edit finances |
| **Secretary** | edit club data, see and maintain qualifications; no finances |
| **Trainer** | view everything like "Read Only"; additionally record attendance for their own groups and sections |
| **Read Only** | view data, change nothing; no finances |

"Club data" means members, club structure, inventory, events, households and tasks.

!!! info "How does Open ClubManager know a trainer's groups?"
    An account with the role **Trainer** is linked to a member by **email address**. The groups and sections of that member decide for which events the trainer may record attendance.

Create users in [user management](../funktionen/benutzer.md).

## Device roles in collaboration

| Area | Board | Treasurer | Secretary | Coach | Equipment manager |
|---|---|---|---|---|---|
| Members | edit | view | edit | view | view |
| Contact details | edit | view | edit | – | – |
| Bank details | edit | edit | – | – | – |
| Finances | edit | edit | – | – | – |
| Inventory | edit | view | view | view | edit |
| Administration | edit | – | – | – | – |

"–" means: the device **does not receive this data at all**. It is transferred encrypted, and the device doesn't have the key. More under [People & devices](../zusammenarbeit/personen-geraete.md).

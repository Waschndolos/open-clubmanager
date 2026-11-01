# Moving to a new computer

## Without collaboration (only on this device)

1. **Old computer:** [create a backup](../funktionen/einstellungen.md#backup) and copy the `.zip` file to a USB stick.
2. **New computer:** [install](../erste-schritte/installation.md) Open ClubManager in the same or a newer version.
3. Run first-run setup with "Only on this device" and create a temporary admin account (**without** demo data).
4. **Settings → Restore backup** → choose the backup → confirm.
5. **Restart** Open ClubManager and log in with your **previous** account.
6. Take samples. If everything is right: **securely delete** the database and backup on the old computer.

!!! tip "Alternative: copy the file"
    You can also copy the database file (`.db`) directly and enter it as the database path during setup on the new computer. Close Open ClubManager on the old computer first.

## With collaboration

With collaboration, a new computer is simply a **new device**:

1. Have the board [invite](../zusammenarbeit/personen-geraete.md#inviting-a-person) you (or, as board, invite yourself).
2. [Join](../zusammenarbeit/beitreten.md) on the new computer. The data comes from the shared folder.
3. **Remove** the old device under [People & devices](../zusammenarbeit/personen-geraete.md#removing-a-device) (reason: "Device is no longer used").

Before removing it, wait until the old device shows "Everything is in sync"; otherwise changes still waiting there are lost.

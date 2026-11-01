# Data protection

Clubs process personal data, often including bank details and data about children. Open ClubManager helps you comply with the GDPR. **The club remains responsible, though.**

!!! note "Not legal advice"
    This page explains how Open ClubManager handles data and which features help you. It is not legal advice. Many sports federations and umbrella organisations provide templates for privacy notices and records of processing activities.

## Where is the data stored?

| Mode | Where | Who has access |
|---|---|---|
| **Only on this device** | in the database file on your computer | whoever has access to your computer and your user account |
| **Collaboration** | on every participating device, plus **encrypted** in the shared folder | the invited devices, each only the areas of its role |

Open ClubManager sends **no** data to the developers or to analytics or advertising services. There is no telemetry.

### Cloud folders in collaboration

In the shared folder, member and bank data is only stored encrypted. The cloud provider does see **technical metadata**, though: file names, file sizes, timestamps, device names, the access list and join requests. Check whether you need a data processing agreement with the provider; with a club-owned Nextcloud this is usually not necessary.

## Protection on the device

The database on the device is **not additionally encrypted**. Protect it by:

- turning on **disk encryption** (BitLocker, FileVault, LUKS)
- locking your computer with a **password**
- protecting **backups** just like the database
- deleting **exports** (Excel, CSV, PDF) after use

## GDPR features

| Obligation | Feature in Open ClubManager |
|---|---|
| **Right of access** (Art. 15) | [data export as PDF](funktionen/mitglieder.md#data-export-gdpr) per member |
| **Data portability** (Art. 20) | data export as JSON |
| **Rectification** (Art. 16) | edit member |
| **Erasure** (Art. 17) | delete member |
| **Access restriction** | [user roles](erste-schritte/rollen.md) and [device roles](zusammenarbeit/personen-geraete.md#roles) |
| **Accountability** | [history](funktionen/verlauf.md) of all changes |
| **Data breach** (Art. 33) | [remove device](zusammenarbeit/personen-geraete.md#removing-a-device) with a note on the 72-hour deadline |
| **Police clearance certificates** (§ 72a SGB VIII, Germany) | [store inspection date only](funktionen/qualifikationen.md#police-clearance-certificates-72a-sgb-viii) |

## Collaboration considerations

Data a device has already received cannot be "taken back", neither by changing its role nor by removing the device. This is true for any software. So only invite people who really need the data for their position, and choose the role with the fewest permissions necessary.

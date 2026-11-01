# FAQ

## General

??? question "Does Open ClubManager cost anything?"
    No. Open ClubManager is free software under the AGPL-3.0 license. There is no paid version, no member limit and no subscription.

??? question "Is there a web version or a mobile app?"
    No, and none is planned. Open ClubManager is a desktop program by design: the data stays on your devices and nobody has to run a server. The downside: without a computer that has Open ClubManager installed, you can't get to the data.

??? question "Can members change their own data?"
    No. Open ClubManager is meant for the club's volunteers, not for the members themselves.

??? question "How many members can Open ClubManager handle?"
    It is designed for clubs with a few dozen to a few thousand members. The database (SQLite) can handle far more.

??? question "Do I need the internet?"
    Only for downloading, updates and collaboration. In "Only on this device" mode you work completely offline.

## Data

??? question "Where is my data?"
    In the database file whose location you chose during setup. You can see the path in the [settings](../funktionen/einstellungen.md).

??? question "Can I put the database file in my Dropbox/OneDrive folder so others can access it?"
    Please don't. Simultaneous synchronisation can corrupt the file, and anyone with access to the folder could read all the data. Use [collaboration](../zusammenarbeit/index.md) instead: it only puts encrypted change files into the folder.

??? question "What happens if the project is discontinued?"
    The installed program keeps working. Your data is in an open SQLite file and can be exported at any time. See [Exporting your data](../migration/daten-mitnehmen.md).

??? question "How do I back up my data?"
    Via **Settings → Data Backup → Back up now**. See [Backup](../funktionen/einstellungen.md#backup).

## Collaboration

??? question "I don't see my colleague's changes."
    Check that your cloud provider's sync program is running and the folder is up to date. Then click **Synchronize now** in the sync status. Other people's data only arrives once your sync program has downloaded it.

??? question "Can I manage several clubs?"
    Not with collaboration at the moment: one club per device is supported. In local mode you can switch between database files in the [settings](../funktionen/einstellungen.md).

??? question "A laptop was stolen. What should I do?"
    Remove the device immediately under [People & devices](../zusammenarbeit/personen-geraete.md#removing-a-device) and check whether you need to notify the data protection authority (72 hours, Art. 33 GDPR).

## Reporting problems

Found a bug or have an idea? Get in touch via the [issue tracker on GitHub](https://github.com/Waschndolos/open-clubmanager/issues). Please **don't include real member data**.

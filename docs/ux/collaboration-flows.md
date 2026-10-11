# UX concept: collaboration flows

- **Status:** Draft
- **Based on:** [ADR 0001](../adr/0001-collaboration-and-sync-architecture.md),
  [sync protocol](../sync-protocol.md)
- **Audience:** contributors building the UI for collaboration mode

The sync protocol is technically demanding. The people using it are board
members of small clubs, often volunteers without IT experience who use the
app a few times a month. This document describes how they create a club,
invite others, handle problems and recover, without ever having to
understand keys, scopes or epochs.

UI texts are given in German, the primary language of the app. They go into
`frontend/src/i18n/de.json`; English and French follow the same meaning.

## Contents

1. [Principles](#1-principles)
2. [Vocabulary](#2-vocabulary)
3. [Where things live](#3-where-things-live)
4. [F1 First start](#f1-first-start)
5. [F2 Creating a club for collaboration](#f2-creating-a-club-for-collaboration)
6. [F3 Inviting a person](#f3-inviting-a-person)
7. [F4 Joining a club](#f4-joining-a-club)
8. [F5 Approving a join request](#f5-approving-a-join-request)
9. [F6 Changing a role, removing a device](#f6-changing-a-role-removing-a-device)
10. [F7 Sync status and "Sync now"](#f7-sync-status-and-sync-now)
11. [F8 Resolving duplicates](#f8-resolving-duplicates)
12. [F9 Warnings](#f9-warnings)
13. [F10 What a role cannot see or change](#f10-what-a-role-cannot-see-or-change)
14. [F11 Deleting personal data for good](#f11-deleting-personal-data-for-good)
15. [F12 Recovery with the emergency key](#f12-recovery-with-the-emergency-key)
16. [F13 Leaving collaboration](#f13-leaving-collaboration)
17. [Implementation notes](#implementation-notes)
18. [Open UX questions](#open-ux-questions)

---

## 1. Principles

1. **No technical terms in the UI.** Words like key, scope, epoch, ACL, HLC,
   segment, snapshot or hash never appear. [§2](#2-vocabulary) defines what we
   say instead.
2. **Local work never blocks.** Sync problems are shown, but people can
   always keep working on their own device. No modal dialogs for sync events.
3. **Explain the why in one sentence.** Every security step (emergency key,
   safety code) says why it matters, in plain words.
4. **Safe defaults, few choices.** Defaults are the recommended option. A
   choice is only offered where the club really has to decide.
5. **One main action per screen.** Secondary actions are text buttons.
6. **Nothing irreversible without a clear confirmation**, and the
   confirmation says what happens and what does not (e.g. "Was diese Person
   bereits gesehen hat, bleibt auf ihrem Gerät").
7. **Problems name the person, not the device id.** "Das Gerät von Erika
   Mustermann (Laptop)" instead of `abcd…`.
8. **Waiting is normal.** Sync goes through a cloud folder and can take
   minutes. Screens that wait say so and let people close the app.

## 2. Vocabulary

| Technical term (spec) | UI term (de) | Notes |
|---|---|---|
| Club with sync | **Zusammenarbeit** / „Verein gemeinsam verwalten“ | |
| Local mode | „Nur auf diesem Gerät“ | |
| Transport / sync root | **Speicherort** | „Gemeinsamer Ordner“ or „Server“ |
| Device | **Gerät** | Always with person and device name: „Erika – Laptop“ |
| Role | **Rolle** | Vorstand, Kassierer, Schriftführer, Übungsleiter, Gerätewart |
| Scope | **Bereich** | Only in role descriptions: „Mitglieder“, „Kontaktdaten“, „Bankdaten“, „Finanzen“, „Inventar“, „Verwaltung“ |
| `r` / `rw` | „sehen“ / „bearbeiten“ | |
| Root key / recovery kit | **Notfallschlüssel** | |
| Invite code | **Einladungscode** | |
| Join request | **Beitrittsanfrage** | |
| Safety code | **Sicherheitscode** | 6 digits, shown as `482 913` |
| Revocation | „Gerät entfernen“ | |
| Unique-value conflict | **Klärungsbedarf** | |
| Schema version too new | „Update nötig“ | |
| Rejected / quarantined segments | „Ungültige Änderungen abgelehnt“ | Admins only |
| Purge | „Endgültig löschen“ | |

## 3. Where things live

- **Header:**
  - **Sync status icon** next to the notification bell ([F7](#f7-sync-status-and-sync-now)).
  - The **notification bell** (`NotificationBell.tsx`) also shows join
    requests, duplicates and warnings, each with a link to where it is solved.
- **Settings → new section „Zusammenarbeit“** with these tabs:
  - **Übersicht:** mode, storage location, last sync, „Jetzt synchronisieren“.
  - **Personen & Geräte:** list of people and their devices, roles, pending
    requests.
  - **Einladungen:** open invites, „Person einladen“.
  - **Rollen:** what each role may see and change (read-only in v1; editable
    later).
  - **Notfall:** reprint the emergency key reminder, „Notfallschlüssel
    verwenden“.
- **Klärungsbedarf:** a list page reached from the bell and the status
  popover ([F8](#f8-resolving-duplicates)).

Only admin devices (role Vorstand) see the tabs Personen & Geräte (editing),
Einladungen and Rollen (editing). Everyone sees Übersicht and their own role.

## F1 First start

Replaces the database-path step of today's setup (`Setup.tsx`). The first
screen asks how the club will be managed:

```
┌──────────────────────────────────────────────────────────────┐
│  Willkommen bei Open ClubManager                             │
│                                                              │
│  Wie möchtest du den Verein verwalten?                       │
│                                                              │
│  ( ) Nur auf diesem Gerät                                    │
│      Für dich allein. Du kannst die Zusammenarbeit später    │
│      jederzeit einschalten.                                  │
│                                                              │
│  ( ) Gemeinsam mit anderen – neuen Verein einrichten         │
│      Du bist im Vorstand und richtest den Verein ein.        │
│                                                              │
│  ( ) Gemeinsam mit anderen – ich habe eine Einladung         │
│      Jemand aus dem Verein hat dir einen Einladungscode      │
│      geschickt.                                              │
│                                                              │
│  [ Optional: Beispieldaten laden ]                           │
│                                                     [Weiter] │
└──────────────────────────────────────────────────────────────┘
```

- **Nur auf diesem Gerät:** continue with today's admin user setup. The
  database lives in the app data directory. There is no path field anymore.
- **Neuen Verein einrichten:** [F2](#f2-creating-a-club-for-collaboration).
- **Ich habe eine Einladung:** [F4](#f4-joining-a-club).
- Demo data is only offered for „Nur auf diesem Gerät“. A club must not
  start with fake members that then sync to everyone.

## F2 Creating a club for collaboration

Reachable from [F1](#f1-first-start) and later from Settings →
Zusammenarbeit → „Zusammenarbeit einschalten“ (with existing local data).

Stepper with 4 steps:

### Step 1 – Speicherort

```
Wo sollen die gemeinsamen Daten liegen?

( ) Gemeinsamer Ordner (empfohlen)
    Ein Ordner, den euer Verein schon nutzt: Nextcloud, OneDrive,
    Dropbox oder Google Drive. Die Daten werden verschlüsselt –
    der Anbieter kann sie nicht lesen.
    Gefunden:  ( ) ~/Nextcloud   ( ) ~/OneDrive
               ( ) Anderen Ordner angeben …
                   [ \\vereinsserver\vorstand        ] [Durchsuchen …]

( ) Nextcloud direkt (ohne Sync-Programm)
    Adresse: [https://cloud.beispiel.de/remote.php/dav/files/…]
    Benutzer: [        ]  App-Passwort: [        ]  [Wie geht das?]

( ) Eigener Server
    Adresse: [https://verein.beispiel.de]
```

- **Folder suggestions:** the app looks for typical sync folders
  (`~/Nextcloud`, `~/OneDrive*`, `~/Dropbox`, `~/Google Drive`) and offers
  them.
- **Own folder:** „Anderen Ordner angeben …“ is always available. The user
  can pick a folder with the system dialog („Durchsuchen …“) or type or
  paste a path, including network paths such as `\\vereinsserver\vorstand`
  or a mounted share like `/Volumes/Verein`. Typed paths are checked as
  soon as the field loses focus.
- **Checks after choosing:**
  1. The folder exists and is writable (test file is created and removed).
     Otherwise: „Auf diesen Ordner kann Open ClubManager nicht schreiben.
     Prüfe den Pfad und deine Berechtigung.“
  2. The app tells the user what kind of folder it found:
     - **Known sync folder** (Nextcloud, OneDrive, …): „✓ Wird von
       Nextcloud synchronisiert.“
     - **Network drive** (UNC path or mounted share): „ℹ Netzlaufwerk. Das
       funktioniert, wenn alle aus dem Vorstand darauf zugreifen können,
       auch von zu Hause, z. B. über VPN.“
     - **Local folder that is not synced:** „⚠ Dieser Ordner scheint nicht
       synchronisiert zu werden. Andere können dann nicht auf die Daten
       zugreifen.“
     The user can continue in every case. A network drive is safe for the
     protocol because files are only ever created, never changed.
  3. The app creates a subfolder `Open ClubManager – <Vereinsname>`. If it
     already contains a club: „Hier gibt es schon einen Verein. Möchtest du
     ihm beitreten?“ → [F4](#f4-joining-a-club).
- **WebDAV:** a „Verbindung testen“ button. Error messages name the likely
  cause (wrong address, wrong password, no permission).

### Step 2 – Verein und du

- **Vereinsname** (prefilled from settings if present).
- **Dein Name** and **Gerätename** (prefilled from the OS, e.g. „Laptop“).
- Info: „Du bekommst die Rolle **Vorstand** und kannst andere einladen.“

### Step 3 – Notfallschlüssel

```
Dein Notfallschlüssel

Wenn alle Geräte des Vorstands verloren gehen, kommt ihr nur mit diesem
Schlüssel wieder an eure Vereinsdaten. Niemand sonst – auch wir nicht –
kann sie wiederherstellen.

[ PDF speichern ]  [ Drucken ]

Bewahre ihn wie einen Tresorschlüssel auf, z. B. ausgedruckt im
Vereinsordner. Nicht im gemeinsamen Ordner speichern.

Zur Kontrolle: Gib die letzten 4 Zeichen des Schlüssels ein: [____]

[x] Ich habe den Notfallschlüssel ausgedruckt oder sicher gespeichert.
                                                         [Weiter]
```

- The PDF has: club name, date, the key as text and QR code, a short guide
  „So verwendest du den Schlüssel“ and a field for a signature or storage
  place.
- „Weiter“ is enabled only after the 4 characters match and the box is
  checked.
- The app warns if the PDF is saved inside the shared folder.
- After this step the key is deleted from the device
  ([spec §5.2](../sync-protocol.md#52-root-key-and-recovery-kit)). The
  screen says so: „Der Schlüssel wird jetzt von diesem Gerät gelöscht.“

### Step 4 – Fertig

- Progress: „Vereinsdaten werden verschlüsselt und im Speicherort abgelegt …“
  (writes the initial snapshot, spec §15.4).
- Then: „Fertig! Als Nächstes: Lade die anderen aus dem Vorstand ein.“ with
  the button **„Person einladen“** ([F3](#f3-inviting-a-person)).

## F3 Inviting a person

Settings → Zusammenarbeit → Einladungen → „Person einladen“.

### Step 1 – Wer und welche Rolle

- **Name** of the person.
- **Rolle:** radio list. Each role has a one-line description and a small
  table showing what it may see and change:

```
( ) Kassierer
    Bankdaten und Finanzen bearbeiten, Mitglieder und Kontakte sehen.

    Mitglieder  Kontaktdaten  Bankdaten  Finanzen  Inventar  Verwaltung
    sehen       sehen         bearbeiten bearbeiten sehen     –
```

### Step 2 – Zugang zum Speicherort

Only for „Gemeinsamer Ordner“ and „Nextcloud direkt“:

```
Gib Erika Zugriff auf den gemeinsamen Ordner

Das geht nicht über Open ClubManager, sondern bei eurem Anbieter.

[Nextcloud] [OneDrive] [Dropbox] [Google Drive]
 1. Öffne Nextcloud im Browser …   (text steps, no screenshots)
 2. …

[x] Erledigt – Erika hat Zugriff auf den Ordner
```

- The provider tab is preselected based on the storage location.
- Guides are short **text steps without screenshots**, plus a link to the
  provider's own help page. Screenshots go out of date with every provider
  UI change and nobody can maintain them reliably.
- The guide shows the folder name to share.
- For „Eigener Server“ this step is skipped.

### Step 3 – Einladungscode

```
Einladung für Erika Mustermann (Kassierer)

  [QR-Code]     7K2P-QX3M-4TFA-W3RB-NC5H-Y6LD     [Kopieren]

Schicke Erika diesen Code, z. B. per E-Mail oder Messenger.
Er ist 7 Tage gültig und kann nur einmal verwendet werden.

Wichtig: Wenn Erika beitritt, nennt sie dir einen 6-stelligen
Sicherheitscode – am besten am Telefon. Erst dann bekommt sie Zugriff.

[Per E-Mail senden]  [Fertig]
```

- **„Per E-Mail senden“** opens the mail program (`mailto:`) with a prepared
  text: what Open ClubManager is, a download link, the code, and „Ich rufe
  dich an, um den Sicherheitscode abzugleichen.“
- The open invite appears under Einladungen with its expiry date and
  „Zurückziehen“.

## F4 Joining a club

From [F1](#f1-first-start) → „Ich habe eine Einladung“.

1. **Code eingeben:** a text field for pasting the code, or „QR-Code scannen“
   (camera, where available). Invalid or expired codes get a clear error:
   „Dieser Code ist abgelaufen. Bitte frag nach einer neuen Einladung.“
2. **Speicherort:**
   - The code only identifies the invite. The app asks where the club's data
     is: „Gemeinsamer Ordner“, „Nextcloud direkt“ or „Server“. A QR code or
     link for WebDAV/server already contains the address.
   - For a folder: „Wähle den Ordner aus, den <Einladende Person> mit dir
     geteilt hat.“ The app offers detected folders that contain a club, and
     „Anderen Ordner angeben …“ with a path field and „Durchsuchen …“, as in
     [F2](#f2-creating-a-club-for-collaboration).
   - When a folder is chosen, the app shows the club name from its
     `club.json`: „✓ Enthält den Verein ‚TSV Beispielstadt e.V.‘“. The person
     confirms that this is the right club.
   - If the folder is not there yet: „Der Ordner ist noch nicht auf diesem
     Gerät angekommen. Prüfe, ob du die Freigabe angenommen hast und dein
     Sync-Programm läuft.“
3. **Dein Name und Gerätename.**
4. **Existing local data** (only if this device already has data, spec §17.1):
   ```
   Auf diesem Gerät gibt es schon Vereinsdaten.
   Beim Beitritt werden sie durch die Daten des Vereins ersetzt.
   [Sicherung speichern und fortfahren]   [Abbrechen]
   ```
   No „continue without backup“ option.
5. **Sicherheitscode:**
   ```
   Fast geschafft!

   Nenne <Einladende Person> diesen Code, am besten am Telefon:

          482 913

   Erst wenn sie ihn bestätigt, bekommst du Zugriff.
   Du kannst die App solange schließen.
   ```
6. **Waiting:** the same screen with „Warte auf Bestätigung …“. When
   approved: progress „Vereinsdaten werden geladen …“, then the dashboard
   with a welcome message naming the role.
7. **Rejected:** „Deine Anfrage wurde abgelehnt. Bitte wende dich an
   <Einladende Person>.“

## F5 Approving a join request

The admin gets a notification „Erika Mustermann möchte beitreten“. Clicking
it opens:

```
Beitrittsanfrage

Erika Mustermann – Laptop
Eingeladen am 11.10.2026 als Kassierer

Erika sieht auf ihrem Bildschirm einen Sicherheitscode.
Frag sie danach und gib ihn hier ein:

  [___ ___]

Stimmt der Code nicht, lehne die Anfrage ab – dann hat
jemand anderes die Einladung benutzt.

[Ablehnen]                                    [Zugriff geben]
```

- The admin **types** the code the new member reads out. Typing forces a
  real comparison; with two codes shown side by side people tend to just
  click „OK“.
- A wrong code → „Der Code stimmt nicht.“ After 3 wrong attempts the request
  is rejected and the invite is withdrawn, with a hint to create a new one.
- „Zugriff geben“ creates the new ACL version, wraps the keys and deletes
  the request (spec §17.2).
- Requests without a matching open invite are not shown to the admin. They
  are ignored silently, as the HMAC check fails.

## F6 Changing a role, removing a device

Settings → Zusammenarbeit → Personen & Geräte:

```
Erika Mustermann           Kassierer         [Rolle ändern] [⋯]
   Laptop                  zuletzt aktiv: heute 14:02
   Handy                   zuletzt aktiv: vor 3 Tagen
Max Muster                 Übungsleiter      [Rolle ändern] [⋯]
   PC                      zuletzt aktiv: vor 2 Wochen
```

„Zuletzt aktiv“ is the time of that device's newest segment.

### Changing a role

- Same role list as in [F3](#f3-inviting-a-person).
- If the new role may see less:
  „Max kann neue Bankdaten ab sofort nicht mehr sehen. Was er bereits gesehen
  hat, bleibt auf seinem Gerät gespeichert.“
- Changing a role applies to all devices of the person.

### Removing a device or a person

Menu `⋯` → „Gerät entfernen“ / „Person entfernen“, with the reason as a
choice. The reason only changes the text:
- „Gerät verloren oder gestohlen“
- „Person scheidet aus dem Amt aus“
- „Gerät wird nicht mehr genutzt“

Confirmation:
```
Gerät „Laptop“ von Erika Mustermann entfernen?

• Das Gerät erhält keine neuen Daten mehr.
• Änderungen, die es ab jetzt macht, werden von allen ignoriert.
• Was bereits auf dem Gerät gespeichert ist, kann nicht gelöscht werden.
  Bei Verlust oder Diebstahl kann das eine Datenpanne sein, die
  innerhalb von 72 Stunden gemeldet werden muss (Art. 33 DSGVO).

[Abbrechen]                                   [Gerät entfernen]
```

- **The last admin device cannot be removed.** The button is disabled with
  the hint „Es muss immer mindestens ein Gerät mit der Rolle Vorstand geben.“
- Removing **one's own device** is only possible via [F13](#f13-leaving-collaboration).
- For „Gerät verloren oder gestohlen“ the app links to a short help page on
  what the club has to do about a possible data breach.

## F7 Sync status and „Sync now“

An icon in the header, next to the bell:

| State | Icon | Tooltip |
|---|---|---|
| Up to date | cloud with check mark | „Alles synchronisiert – zuletzt 14:02“ |
| Syncing | cloud with arrows (animated) | „Wird synchronisiert …“ |
| Offline / storage unreachable | cloud with slash | „Speicherort nicht erreichbar – 3 Änderungen warten. Du kannst normal weiterarbeiten.“ |
| Attention | cloud with exclamation mark (warning color) | „Bitte ansehen“ |

Clicking the icon opens a popover:

```
Zusammenarbeit
✓ Zuletzt synchronisiert: heute, 14:02
  Speicherort: Nextcloud – Open ClubManager – TSV Beispielstadt
  Wartende eigene Änderungen: 0

⚠ 2× Klärungsbedarf                    [Ansehen]

[Jetzt synchronisieren]
```

- **„Jetzt synchronisieren“** flushes the outbox and checks the storage
  immediately (spec §23.2). For a cloud folder it adds: „Neue Daten anderer
  kommen erst an, wenn dein Sync-Programm sie heruntergeladen hat.“
- **Local mode:** no icon.

## F8 Resolving duplicates

A „Klärungsbedarf“ page lists the conflicts from spec §14.2. Each is a card:

```
Mitgliedsnummer 1042 ist doppelt vergeben

  Erika Mustermann            Paul Beispiel
  angelegt 03.10. von         angelegt 03.10. von
  Erika (Laptop)              Max (PC)
  behält die Nummer           ⚠ Nummer muss geändert werden

  [Paul die nächste freie Nummer geben (1057)]   [Nummer selbst wählen]
```

- The card follows the spec: the earlier assignment keeps the value, the
  other row is flagged.
- **Duplicate name** (role, group or section): actions „Zusammenführen“
  (merge both into the one that keeps the name) and „Umbenennen“.
- **Duplicate e-mail** of two members: „E-Mail bei Paul ändern“, and a hint
  that this may be the same person entered twice, with a link to both
  members.
- People without write permission for the area see the card read-only:
  „Das kann jemand mit der Rolle Vorstand oder Schriftführer klären.“
- Flagged rows show a small warning icon in tables (e.g. the member list),
  linking to the card.

## F9 Warnings

Shown as notifications and in the status popover, never as blocking dialogs.

| Situation (spec) | Who sees it | Text (de) | Action |
|---|---|---|---|
| Newer data format from another device (§16) | everyone | „Ein anderes Gerät nutzt eine neuere Version von Open ClubManager. Aktualisiere die App, um dessen Änderungen zu sehen. Deine eigenen Änderungen werden weiter gespeichert.“ | „Update suchen“ |
| `min_schema` above own version (§16) | everyone | „Der Vorstand hat ein Update vorausgesetzt. Bis dahin kannst du Daten nur ansehen.“ | „Update suchen“ |
| Clock far ahead on another device (§12) | admins | „Die Uhrzeit auf dem Gerät von Max (PC) geht deutlich vor. Seine Änderungen können dadurch andere überschreiben. Bitte Max, die Uhrzeit zu prüfen.“ | „Gerät ansehen“ |
| Own clock wrong (detected against storage file times) | the person | „Die Uhrzeit dieses Geräts scheint falsch zu sein. Bitte stelle sie auf automatisch.“ | – |
| Rejected or quarantined changes (§13.2) | admins | „Ungültige Änderungen vom Gerät ‚PC‘ von Max wurden abgelehnt. Das kann ein Übertragungsfehler oder ein Manipulationsversuch sein.“ | „Details“, „Gerät entfernen“ |
| Waiting for missing files (gap) longer than 1 day | everyone | „Es fehlen Daten vom Gerät ‚Laptop‘ von Erika. Vermutlich ist ihr Sync-Programm nicht fertig.“ | – |
| Storage not reachable longer than 7 days | everyone | „Seit 7 Tagen keine Verbindung zum Speicherort. Deine Änderungen sind nur auf diesem Gerät.“ | „Speicherort prüfen“ |

## F10 What a role cannot see or change

- **Areas a role cannot see are not shown at all.**
  - The member dialog of a Übungsleiter has no tab „Bankdaten“ and no
    contact fields.
  - Finance is missing from the menu.
  - No „Keine Berechtigung“ placeholders: they would only raise questions.
- **Areas a role can only see** are shown with disabled inputs, a lock icon
  and the tooltip „Nur lesen – deine Rolle: Kassierer“. Buttons like „Neues
  Mitglied“ are hidden.
- **Settings → Zusammenarbeit → Übersicht** shows „Deine Rolle“ with the
  same see/change table as in [F3](#f3-inviting-a-person).
- **Deleting a member** in an area-restricted role
  (spec §14.3): „Kontaktdaten und Bankdaten dieses Mitglieds kann nur der
  Vorstand löschen. Sie werden ihm zur Bereinigung angezeigt.“ Admins see
  a „Verwaiste Daten“ section on the Klärungsbedarf page.

## F11 Deleting personal data for good

Members are first archived or deleted as today (tombstone). Separately, the
member detail page offers „Endgültig löschen“ (purge, spec §15.3), for
admins and for deleted members only:

```
Daten von Paul Beispiel endgültig löschen?

Die Daten werden auf allen Geräten gelöscht und können nicht
wiederhergestellt werden.

⚠ Paul hat 12 Buchungen in den Finanzen. Buchungsbelege müssen
  in der Regel 10 Jahre aufbewahrt werden (§ 147 AO).
  ( ) Nur Kontakt- und Bankdaten löschen, Buchungen behalten (empfohlen)
  ( ) Alles löschen

Hinweis: Kopien im Papierkorb oder in älteren Dateiversionen eures
Cloud-Speichers kann Open ClubManager nicht löschen. Sie sind
verschlüsselt, werden aber erst entfernt, wenn ihr den Papierkorb
leert.

Zur Bestätigung den Namen eingeben: [               ]
[Abbrechen]                                    [Endgültig löschen]
```

## F12 Recovery with the emergency key

Reachable from the first-start screen (small link „Notfallschlüssel
verwenden“) and from Settings → Zusammenarbeit → Notfall.

1. Explain when to use it: „Nur verwenden, wenn kein Gerät mit der Rolle
   Vorstand mehr verfügbar ist.“
2. Enter or scan the key, choose the storage location (as in [F4](#f4-joining-a-club)).
3. Name and device name. This device becomes a Vorstand device.
4. Afterwards: „Du bist jetzt Vorstand. Entferne Geräte, die verloren
   gegangen sind“, with a link to [F6](#f6-changing-a-role-removing-a-device).
5. Reminder: „Der Notfallschlüssel bleibt gültig. Bewahre ihn wieder sicher
   auf.“

## F13 Leaving collaboration

Settings → Zusammenarbeit → „Dieses Gerät abmelden“.

- Choice: „Daten auf diesem Gerät behalten (nur lesen, nicht mehr
  synchronisiert)“ or „Daten von diesem Gerät löschen“.
- The app does not send a request. It tells the person: „Bitte jemanden aus
  dem Vorstand, dein Gerät unter Personen & Geräte zu entfernen. Bis dahin
  bleibt es in der Liste.“
- The last admin cannot leave. The hint is to give the Vorstand role to
  someone else first.

## Implementation notes

- **Routes:**
  - `/setup` gets the new first step ([F1](#f1-first-start)) and the
    sub-flows F2, F4 and F12 as stepper variants.
  - New routes: `/settings/collaboration/*` and `/conflicts`.
- **Components:**
  - Reuse MUI `Stepper` as in `Setup.tsx`.
  - Sync status: a new `SyncStatusButton` in the header next to
    `NotificationBell`.
  - Notifications need types (join request, conflict, warning) with a link.
    Today `NotificationContext` only holds texts.
- **Role-aware UI:** a hook like `usePermission(area)` returning `hidden`,
  `read` or `write`, used by menu, pages and dialogs. The backend must still
  check every request; the UI only hides.
- **API:**
  - Needed endpoints (to be added to `openapi.yaml`): sync status, sync now,
    create club, invites, join, requests and approval, devices, roles,
    revoke, conflicts, recovery.
  - Sync progress via polling of the status endpoint (every 2 s while a flow
    is running).
- **Emergency key PDF:** generated in the frontend or the backend. It must
  not be written to disk except where the user saves it.
- **Safety code:** large monospace font, grouped as `482 913`, readable
  aloud. Screen readers read digits one by one.
- **Tauri:**
  - Folder picker and detecting sync folders through Tauri's dialog and fs
    APIs.
  - Camera for QR scanning is optional. Pasting the code must always work.
- **i18n:** all texts in `de.json`, `en.json`, `fr.json`. The provider guides
  (F3 step 2) are longer texts with images and should live as separate
  Markdown files per language.

## Decisions

Decided after the first draft:

1. **Short invite code.** The code contains only the invite id and secret
   (24 characters, e.g. `7K2P-QX3M-4TFA-W3RB-NC5H-Y6LD`). The club is found
   through the storage location; for WebDAV and server a QR code or link
   carries the address. Spec §17.2 is updated accordingly.
2. **One club per device in v1.** If a device that already belongs to a club
   uses an invite, the app explains: „Auf diesem Gerät ist schon ein Verein
   eingerichtet. Mehrere Vereine werden noch nicht unterstützt.“
3. **Only the five default roles in v1.** The ACL format already supports
   custom roles; the editor comes later. The Rollen tab is read-only.
4. **Desktop only in v1.** Mobile access comes later via the browser in
   server mode. Cloud-folder sync on phones is not reliable enough.
5. **Provider guides as text without screenshots**, with a link to the
   provider's help (see [F3](#f3-inviting-a-person)).
6. **No leave request.** [F13](#f13-leaving-collaboration) asks the person
   to have a Vorstand member remove the device. The spec stays unchanged.

---
routes:
  - /collaboration/recover
---

# Emergency key

The emergency key is generated once when you [set up the club](einrichten.md). With it, you regain access to your club data and administration when **no device with the board role** is available any more, for example because the only board laptop was stolen.

## Keeping it safe

- Keep it like a **safe key**: printed in the club's files, in a bank deposit box, at the chair's home.
- **Don't** store it in the shared cloud folder: whoever has the folder and the emergency key has full access.
- Nobody else can restore it, not even the Open ClubManager developers.

## Using it

Only use it when there really is no board device left.

1. Install Open ClubManager on a device and start setup with **Use the emergency key** (or under **Collaboration → Emergency**).
2. Paste or type the whole text of the emergency key. The lines `club-id`, `root-sign-key`, `root-age-key` and `check` are the important ones.
3. Click **Restore club**.

This device is then a **board device**. Next, remove all lost devices under [People & devices](personen-geraete.md#removing-a-device).

The emergency key stays valid; keep it safe again.

## Common errors

| Message | Meaning |
|---|---|
| "The emergency key is incomplete or contains a typo" | check the text, especially the `check` line |
| "This emergency key belongs to a different club" | wrong key or wrong folder |

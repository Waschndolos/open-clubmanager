# Open ClubManager

[![CI](https://github.com/Waschndolos/open-clubmanager/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Waschndolos/open-clubmanager/actions/workflows/ci.yml)

🚧 This project is currently under construction. Features and structure may change frequently. 🚧

**Open ClubManager** is an open source web application for managing clubs. It features a modern React-based frontend and
a TypeScript + SQLite backend powered by Prisma and Express. ClubManager is designed to be lightweight, self-hostable,
and suitable for small to medium-sized clubs or associations.

---
## 💡 Idea of this project
To contribute meaningfully to this project, it helps to understand the motivation behind it.

In Germany — and likely many other places — small clubs or "Vereine" often manage their members, finances, and activities using Excel spreadsheets or Word documents. That works — until the club starts growing. Suddenly, more people are involved: a treasurer, a board, maybe a youth coordinator. Everyone needs access to the same information, and coordination becomes painful.

Eventually, the club ends up maintaining dozens of separate lists and files — and the chaos begins. At that point, most clubs start looking for a centralized software solution. The problem? Most of those systems are proprietary and expensive — perfectly fine for large organizations, but a real hurdle for small clubs with limited budgets and no IT team.

That's where Open ClubManager comes in.

This project aims to provide a free, open source, and easy-to-use management tool for small to medium-sized clubs. It's designed to work out of the box — no cloud infrastructure, no complicated database setup.

Instead, Open ClubManager uses a lightweight SQLite database, which can either:

    be stored locally (ideal if you're managing the club on your own), or

    placed on a shared network drive, so that multiple team members (e.g. treasurer, board, secretary) can access and update the same data.

The goal is to reduce complexity and cost — and to empower small clubs to manage themselves better without needing tech expertise or subscriptions.

Open ClubManager is built by volunteers for volunteers.
---

## 🗂️ Folder Mode (No Host)

Folder Mode enables multi-user collaboration via a **shared folder** (SharePoint, OneDrive, Dropbox, SMB/NAS, USB) with **no external backend server** required.

### How it works

Instead of a SQLite database, all club data is stored as an **append-only event log** inside the chosen folder:

```
ClubData/
  club.json
  schema-version.json
  events/
    2026-03-18T12-00-01-123Z_<uuid>.json
    ...
  snapshots/
    snapshot-00000200.json
  locks/
    member-123.lock
```

- **Events** are never overwritten — each change is a new file. Sync tools (OneDrive, Dropbox…) handle new files reliably.
- **State** is reconstructed at runtime by loading the latest snapshot and replaying events on top of it.
- **Locking** prevents two people from editing the same member simultaneously.

### Steps for end users

1. Install and open the **Open ClubManager** desktop app.
2. On first start you will be taken to the **"Select Club Folder"** screen.
3. Click **Select Club Folder** and choose any folder — local, shared network drive, or a folder synced by SharePoint/OneDrive/Dropbox.
4. That's it. The app creates the required folder structure automatically.

Anyone with access to the same folder can open the app and work with the same data.  
No Node.js, no `.env` file, no server setup.

### Locking behaviour

- When a user opens the **Edit Member** dialog, a lock file is created: `locks/member-<id>.lock`.
- The lock file records the owner and an expiry time (default: 10 minutes).
- While the lock is held, other users see the member as locked and cannot edit it.
- If a lock is older than its expiry time it is considered **stale** and is automatically released when another user tries to acquire it.
- Users can also manually force-release a stale lock.

### Backward compatibility

- `npm run dev:browser` continues to use the existing Express/Prisma backend unchanged.
- `npm run dev:electron` starts the backend alongside Electron (development convenience).
- In a production Electron build no backend process is required — the app uses IPC to its own built-in storage layer.

---



* 🧑‍🤝‍🧑 Manage members, roles, groups, and sections
* 📄 Track addresses, banking information, SEPA mandates
* 🗓️ Record entry/exit dates, comments, and status
* 🔒 Local database (SQLite) – no cloud dependency
* 🎨 React + TypeScript frontend
* ⚙️ Express + Prisma backend
* 🌱 Seed system for demo data in development

---

## 📦 Tech Stack

* **Frontend**: React, TypeScript, Axios, Vite
* **Backend**: Node.js, Express, TypeScript
* **Database**: SQLite (via Prisma ORM)
* **Dev Tools**: ts-node, Prisma Studio, WebStorm/VS Code

---

## 📁 Project Structure

```
/clubmanager
├── assets/          # Assets for the electron build
├── backend/         # Express API + Prisma + SQLite
│   ├── src/
│   │   ├── server.ts
│   │   └── routes/
│   ├── prisma/
│   │   ├── schema.prisma
│   │   ├── migrations/
│   │   └── seed.ts
├── electron/        # Electron build
├── frontend/        # React client
```

---

## 🛠️ Getting Started

### Prerequisites

* Node.js 20
* npm

### ⚡ Quick Setup (recommended)

> **One-command setup** – installs all dependencies and initialises the development database in one step.

First, create a file named `.env` in the `backend` directory with the following content:
```plaintext
# The url where the backend will find the SQLite database
DATABASE_URL="file:/home/myUser/clubmanager.db"

# Alternatively, for users who use "Please Reboot OS" you can use:
# DATABASE_URL="file:C:\\Users\\myUser\\clubmanager.db"
```

Then, from the project root run:

```bash
npm run setup
```

This will:
1. Install all dependencies (root, frontend, backend)
2. Run Prisma migrations
3. Seed the database with demo data

An admin user `admin@admin.com` / `admin` is created automatically.

Once setup is complete, start the development servers:

```bash
# Start frontend + backend in the browser
npm run dev:browser

# or start frontend + backend + Electron
npm run dev:electron
```

---

### Manual Setup

If you prefer to set up each part individually:

#### Backend

Create a file named `.env` in the `backend` directory with the following content:
```plaintext
# The url where the backend will find the SQLite database
DATABASE_URL="file:/home/myUser/clubmanager.db"

# Alternatively, for users who use "Please Reboot OS" you can use:
# DATABASE_URL="file:C:\\Users\\myUser\\clubmanager.db"
```

```bash
cd backend
npm install
npm run setup      # runs migrations + seeds the database
npm run dev        # start the backend server
```

#### Frontend

```bash
cd frontend
npm install
npm run dev        # start the Vite dev server
```

### Start Frontend and Backend in Dev mode
```bash
cd <rootProject>
# Start the frontend and backend in browser
npm run dev:browser
# or start the frontend and backend in electron
npm run dev:electron
```

The initial user for the development mode is:
```plaintext
Email:      admin@admin.com
Password:   admin
```

---

## 🌱 Development Mode

When starting the backend in development mode (`NODE_ENV=development`), seed data will be automatically inserted to
simulate real-world use cases.

---

## 📋 Available npm Scripts

All scripts below can be run from the **project root**.

| Script | Description |
|---|---|
| `npm run setup` | Install all dependencies and initialise the dev database (first-time setup) |
| `npm run install:all` | Install dependencies for root, frontend, and backend |
| `npm run dev:browser` | Start frontend + backend in browser dev mode |
| `npm run dev:electron` | Start frontend + backend + Electron in dev mode |
| `npm run build` | Build frontend, backend, and Electron |
| `npm run lint` | Lint frontend and backend source code |
| `npm run prisma:studio` | Open Prisma Studio (database GUI) |
| `npm run prisma:seed` | Re-seed the development database |
| `npm run prisma:reset` | Reset and re-run all migrations |
| `npm run dist` | Create a distributable Electron package |
| `npm run licenses` | Regenerate third-party licence files |

---

## 🧪 Testing

You can use Postman or any REST client to test:

* `GET /api/members`
* `POST /api/members`
* `PUT /api/members/:id`
* `DELETE /api/members/:id`

Prisma Studio is available via:

```bash
# From the project root
npm run prisma:studio

# or from the backend directory
npx prisma studio
```

---

## 🤝 Contributing

Contributions, issues and feature requests are welcome!

For details, see [CONTRIBUTING.md](CONTRIBUTING.md).


Please follow the [Code of Conduct](#-code-of-conduct) and submit only clean, tested code.

---

## 📜 Code of Conduct

This project and everyone participating in it is expected to adhere to
the [Contributor Covenant](CODE_OF_CONDUCT.md).

Be respectful, inclusive, and constructive. We value every contribution, regardless of experience level, background, or
role.

---

## 📄 License

This project is licensed under the Creative Commons Attribution-NonCommercial 4.0 International License (CC BY-NC 4.0).

You may use, modify, and share the code non-commercially, provided you give appropriate credit. Commercial use is not permitted.

For full terms, see the LICENSE file or visit https://creativecommons.org/licenses/by-nc/4.0/.

---

## 📚 Licenses of Used Libraries

This project uses open source packages such as React, Express, and Prisma. These components are licensed under permissive licenses (MIT, Apache-2.0, etc.). You can find their license texts in their respective repositories or via `npm`.

The original code of ClubManager is licensed under **CC BY-NC 4.0**.

For a list of all third-party libraries and their licenses, see 
* [backend/THIRD_PARTY_LICENSES.md](backend/THIRD_PARTY_LICENSES.md).
* [frontend/THIRD_PARTY_LICENSES.md](frontend/THIRD_PARTY_LICENSES.md)

## 💬 Questions & Feedback

If you have questions, ideas, or feedback, feel free to open an issue or start a discussion.

Let's make club management easier, together! 💙

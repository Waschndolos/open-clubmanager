# Open ClubManager

[![CI](https://github.com/Waschndolos/open-clubmanager/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Waschndolos/open-clubmanager/actions/workflows/ci.yml)

🚧 This project is currently under construction. Features and structure may change frequently. 🚧

**Open ClubManager** is an open source web application for managing clubs. It features a modern React-based frontend and
a Go + Chi backend API. ClubManager is designed to be lightweight, self-hostable,
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

## 🗄️ Database-Only Architecture

Open ClubManager now runs exclusively with the backend API and a database.

- `npm run dev:browser` starts frontend + backend.
- `npm run dev:electron` starts frontend + backend + Electron shell.
- In all environments, application data is loaded and persisted through `/api/v2`.

---



* 🧑‍🤝‍🧑 Manage members, roles, groups, and sections
* 📄 Track addresses, banking information, SEPA mandates
* 🗓️ Record entry/exit dates, comments, and status
* 🔒 Local database (SQLite) – no cloud dependency
* 🎨 React + TypeScript frontend
* ⚙️ Go + Chi backend
* 🌱 Seed system for demo data in development

---

## 📦 Tech Stack

* **Frontend**: React, TypeScript, Axios, Vite
* **Backend**: Go, Chi, oapi-codegen
* **Database**: SQLite / MySQL
* **Dev Tools**: go toolchain, npm, WebStorm/VS Code

---

## 📁 Project Structure

```
/clubmanager
├── assets/          # Assets for the electron build
├── server/          # Go API + Chi router
│   ├── cmd/server/  # binary entry point
│   └── internal/    # generated OpenAPI bindings + handlers
├── electron/        # Electron build
├── frontend/        # React client
```

---

## 🛠️ Getting Started

### Prerequisites

* Node.js 20
* npm
* Go 1.24+

### ⚡ Quick Setup (recommended)

> **One-command setup** – installs all dependencies and initialises the development database in one step.

First, create a file named `.env` in the `backend` directory with the following content:
```plaintext
# Default local database URL (SQLite)
DATABASE_URL="file:/home/myUser/clubmanager.db"

# Development secrets for API authentication
JWT_ACCESS_SECRET="dev_access_secret_change_me"
JWT_REFRESH_SECRET="dev_refresh_secret_change_me"

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
3. Seed the database with demo data (members, roles, groups, sections)

If the database contains no users yet, the app will guide you through the setup flow:
1. Choose database mode:
   - local use: `sqlite-local`
   - shared multi-user use: `mysql-shared`
2. Enter a MySQL URL for shared mode
3. Create the first admin account

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
# Default local database URL (SQLite)
DATABASE_URL="file:/home/myUser/clubmanager.db"

# Recommended for local development
JWT_ACCESS_SECRET="dev_access_secret_change_me"
JWT_REFRESH_SECRET="dev_refresh_secret_change_me"

# Alternatively, for users who use "Please Reboot OS" you can use:
# DATABASE_URL="file:C:\\Users\\myUser\\clubmanager.db"
```

```bash
cd backend
npm install
npm run setup      # sqlite: runs migrations + seeds the database
npm run setup:mysql # mysql: pushes schema (requires reachable MySQL DB)
npm run dev        # start the backend server
```

#### Frontend

Create a file named `.env` in the `frontend` directory with the following content:

```plaintext
VITE_APP_VERSION=1.0.0
VITE_BACKEND_ORIGIN=http://localhost:3001
```

```bash
cd frontend
npm install
npm run dev        # start the Vite dev server
```

### API v2 bootstrap endpoints

The backend is exposed via the versioned namespace `/api/v2`.

- `GET /api/v2/system/health`
- `GET /api/v2/system/meta`
- `GET /api/v2/setup/status`
- `POST /api/v2/setup/initialize`
- `POST /api/v2/auth/login`
- `POST /api/v2/auth/refresh-token`
- `POST /api/v2/auth/logout`
- `GET /api/v2/auth/profile`
- `GET /api/v2/members`
- `GET /api/v2/members/:id`
- `POST /api/v2/members`
- `PUT /api/v2/members/:id`
- `DELETE /api/v2/members/:id`

### 📘 API documentation

- OpenAPI spec: [`openapi.yaml`](openapi.yaml)
- ReDoc page: [`docs/index.html`](docs/index.html)

If you update `openapi.yaml`, regenerate the ReDoc page with:

```bash
npm run docs:redoc
```

The repository also contains a publishing workflow in [`.github/workflows/api-docs.yml`](.github/workflows/api-docs.yml).
Pushes to `main` regenerate the ReDoc bundle and publish it to the `gh-pages` branch.

To expose the hosted documentation in GitHub Pages, set the repository Pages source to:

- **Deploy from a branch**
- **Branch:** `gh-pages`
- **Folder:** `/ (root)`

GitHub usually exposes that site under a URL like:

```text
https://waschndolos.github.io/open-clubmanager/
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
No fixed default user is created by the seed script.
If there are no users yet, open the setup flow in the app and create the first admin account.
```

---

## 🌱 Development Mode

Demo data is created by running `npm run setup` or `npm run prisma:seed`.
Starting the backend alone does not automatically reseed the database.

---

## 📋 Available npm Scripts

All scripts below can be run from the **project root**.

| Script | Description |
|---|---|
| `npm run setup` | Install all dependencies for frontend + Go server |
| `npm run install:all` | Install dependencies for root + frontend and run `go mod tidy` in `server/` |
| `npm run dev:browser` | Start frontend + backend in browser dev mode |
| `npm run dev:electron` | Start frontend + backend + Electron in dev mode |
| `npm run build` | Build frontend, backend, and Electron |
| `npm run lint` | Lint frontend and run Go tests for the server |
| `npm run dist` | Create a distributable Electron package |
| `npm run licenses` | Regenerate third-party licence files |

---

## 🧪 Testing

You can use Postman or any REST client to test:

* `GET /api/v2/members`
* `POST /api/v2/members`
* `PUT /api/v2/members/:id`
* `DELETE /api/v2/members/:id`

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

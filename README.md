# Open ClubManager

**Open ClubManager** is an open source web application for managing clubs. It features a modern React-based frontend and
a TypeScript + SQLite backend powered by Prisma and Express. ClubManager is designed to be lightweight, self-hostable,
and suitable for small to medium-sized clubs or associations.

---

## 🚀 Features

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
├── frontend/        # React client
├── backend/         # Express API + Prisma + SQLite
│   ├── src/
│   │   ├── server.ts
│   │   └── routes/
│   ├── prisma/
│   │   ├── schema.prisma
│   │   ├── migrations/
│   │   └── seed.ts
```

---

## 🛠️ Getting Started

### Prerequisites

* Node.js >= 18
* npm or pnpm

### Backend Setup

```bash
cd backend
npm install
npx prisma migrate dev --name init
npm run seed
npm run dev
```

### Frontend Setup

```bash
cd frontend
npm install
npm run dev
```

---

## 🌱 Development Mode

When starting the backend in development mode (`NODE_ENV=development`), seed data will be automatically inserted to
simulate real-world use cases.

---

## 🧪 Testing

You can use Postman or any REST client to test:

* `GET /api/members`
* `POST /api/members`
* `PUT /api/members/:id`
* `DELETE /api/members/:id`

Prisma Studio is available via:

```bash
npx prisma studio
```

---

## 🤝 Contributing

Contributions, issues and feature requests are welcome!

1. Fork this repository
2. Create a new branch (`git checkout -b feature/my-feature`)
3. Commit your changes (`git commit -m 'feat: add new feature'`)
4. Push to the branch (`git push origin feature/my-feature`)
5. Create a pull request

Please follow the [Code of Conduct](#-code-of-conduct) and submit only clean, tested code.

---

## 📜 Code of Conduct

This project and everyone participating in it is expected to adhere to
the [Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct).

Be respectful, inclusive, and constructive. We value every contribution, regardless of experience level, background, or
role.

---

## 📄 License

This project is licensed under the [CC BY-NC 4.0 License](LICENSE).

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

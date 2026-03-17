# Contributing to ClubManager

First of all, thank you for taking the time to contribute to **ClubManager**! 🙏

We welcome contributions of all kinds, including bug reports, feature requests, documentation improvements, and code contributions.

---

## 🧰 Prerequisites

* Node.js >= 18
* npm or pnpm
* Git

---

## 🧪 Running the App Locally

### Quick Setup

From the project root, run a single command to install everything and set up the database:

```bash
npm run setup
```

Then start the dev servers:

```bash
npm run dev:browser
```

### Manual Setup

#### Backend

```bash
cd backend
npm install
npm run setup      # runs Prisma migrations and seeds the database
npm run dev        # start the backend server
```

#### Frontend

```bash
cd frontend
npm install
npm run dev        # start the Vite dev server
```

Backend will be available at `http://localhost:3001`, and frontend at `http://localhost:5173`.

---

## ✅ How to Contribute

### 1. Fork & Clone

* Fork this repository to your GitHub account
* Clone your fork locally

### 2. Create a Branch

```bash
git checkout -b feature/my-feature-name
```

### 3. Make Changes

* Write clear, readable code
* Follow existing code style
* Add comments where helpful
* Update documentation if necessary

### 4. Test Before Commit

* Make sure your feature or fix works as expected
* Run the app and test relevant parts manually

### 5. Commit with a Clear Message

```bash
git commit -m "feat: add new feature X"
```

### 6. Push and Open a Pull Request

```bash
git push origin feature/my-feature-name
```

* Then go to GitHub and open a Pull Request against the `main` branch

---

## 📎 Code Style

* Use TypeScript consistently
* Use 2-space indentation
* Keep functions and components small and focused
* Use meaningful variable and function names

---

## 🚫 Please Avoid

* Changing unrelated files
* Adding large new dependencies without discussion
* Committing directly to the `main` branch
* Submitting untested code

---

## 🛡️ Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). Please be respectful, inclusive, and constructive in your collaboration.

---

Thank you again for helping make ClubManager better! 💙

# Dev Container

This configuration sets up a complete development environment for `open-clubmanager`.

## Included toolchains and system packages

- Node.js 22 + npm
- Go 1.25.0
- Rust 1.77.2 with `cargo`, `clippy`, and `rustfmt`
- Python 3
- SQLite and MySQL CLI tools
- Linux dependencies for Tauri 2 on Ubuntu 24.04

## Automatic bootstrap

The following commands are run automatically when the container is created for the first time:

1. `npm ci`
2. `npm --prefix frontend ci`
3. `npm --prefix db ci`
4. `go mod download`
5. `cargo fetch`
6. `npm run prisma:validate`
7. `npm run tauri:check`

## Notes

- For browser-based development, ports `5173` (frontend) and `3001` (backend) are forwarded.
- To run `npm run dev` with the native Tauri window, additional display forwarding from the host may be required. The necessary build dependencies are already installed in the container.
- If `server/.env` or `frontend/.env` are missing, create them based on the examples in `README.md`.


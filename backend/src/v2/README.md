# API v2 (Start Implementation)

This folder contains the first implementation slice for the backend replacement.

## Goals of this slice

- Introduce a versioned API namespace (`/api/v2`).
- Introduce service/repository separation for new modules.
- Keep existing `/api/*` endpoints functional during migration.

## Available endpoints

- `GET /api/v2/system/health`
- `GET /api/v2/system/meta`
- `GET /api/v2/setup/status`
- `POST /api/v2/setup/initialize`
- `POST /api/v2/auth/login`
- `POST /api/v2/auth/refresh-token`
- `POST /api/v2/auth/logout`
- `GET /api/v2/auth/profile` (Bearer token required)
- `GET /api/v2/members`
- `GET /api/v2/members/:id`
- `POST /api/v2/members` (Bearer token required)
- `PUT /api/v2/members/:id` (Bearer token required, with `expectedVersionToken`)
- `DELETE /api/v2/members/:id` (Bearer token required)

## Notes

- JWT secrets are read from `JWT_ACCESS_SECRET` and `JWT_REFRESH_SECRET`.
- If these are missing, legacy fallback values are used temporarily to avoid breaking existing local setups.
- Fallback usage is visible via `GET /api/v2/system/meta`.


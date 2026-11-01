-- Password reset tokens are stored here instead of in server memory (#184).
-- Only the SHA-256 hash of a token is kept, so a copy of the database does
-- not hand out working reset links. user_id is a users.id or the config-file
-- admin's fixed id (db.AdminUserID), hence no foreign key. The table was
-- never written before, so it is simply recreated.

-- +goose Up
DROP TABLE password_reset_tokens;
CREATE TABLE password_reset_tokens (
    id          TEXT NOT NULL PRIMARY KEY,
    token_hash  TEXT NOT NULL UNIQUE,
    user_id     TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at  TEXT NOT NULL
);
CREATE INDEX password_reset_tokens_user ON password_reset_tokens (user_id);

-- +goose Down
DROP TABLE password_reset_tokens;
CREATE TABLE password_reset_tokens (
    id          TEXT NOT NULL PRIMARY KEY,
    token       TEXT NOT NULL UNIQUE,
    user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at  TEXT NOT NULL
);

-- Refresh tokens keep a login alive past the short-lived access token. The
-- original table was never written to: it could only hold database users,
-- while the administrator from config.json has no row in users. It is
-- rebuilt keyed by e-mail address, with user_id set for database users only
-- so that deleting a user still cascades.
--
-- Only a SHA-256 hash of each token is stored. Tokens are rotated on every
-- refresh: the used token is marked replaced and a new one of the same
-- family is inserted. Presenting a replaced token again revokes the family.
-- app_role and created_at are checked against the account on every refresh,
-- like the claims of an access token.

-- +goose Up
DROP TABLE refresh_tokens;
CREATE TABLE refresh_tokens (
    id           TEXT NOT NULL PRIMARY KEY,
    token_hash   TEXT NOT NULL UNIQUE,
    family_id    TEXT NOT NULL,
    email        TEXT NOT NULL,
    app_role     TEXT NOT NULL,
    user_id      TEXT REFERENCES users (id) ON DELETE CASCADE,
    created_at   TEXT NOT NULL,
    expires_at   TEXT NOT NULL,
    replaced_at  TEXT
);
CREATE INDEX refresh_tokens_family ON refresh_tokens (family_id);
CREATE INDEX refresh_tokens_email ON refresh_tokens (email);
CREATE INDEX refresh_tokens_user ON refresh_tokens (user_id);

-- +goose Down
DROP TABLE refresh_tokens;
CREATE TABLE refresh_tokens (
    id          TEXT NOT NULL PRIMARY KEY,
    token       TEXT NOT NULL UNIQUE,
    user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at  TEXT NOT NULL
);

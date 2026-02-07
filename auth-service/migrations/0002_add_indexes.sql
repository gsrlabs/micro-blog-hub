-- migrations/0002_add_indexes.sql
-- +goose Up
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);

-- +goose Down
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_email;
-- +goose Up
BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS urlshrt(uuid INTEGER, short text, original text primary key, user_id INTEGER);
CREATE INDEX IF NOT EXISTS idx_combined ON urlshrt USING BTREE (uuid, short, original, user_id);
CREATE INDEX IF NOT EXISTS idx_urls ON urlshrt USING BTREE (short, original);
COMMIT;

-- +goose Down
BEGIN TRANSACTION;
DROP TABLE IF EXISTS urlshrt;
COMMIT;
-- +goose Up
CREATE TABLE IF NOT EXISTS urlshrt(uuid INTEGER, short text, original text primary key);
CREATE INDEX IF NOT EXISTS idx_combined ON urlshrt USING BTREE (uuid, short, original);

-- +goose Down
DROP TABLE IF EXISTS urlshrt;
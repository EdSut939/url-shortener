-- +goose Up
ALTER TABLE urls
ADD visits int default 0;

-- +goose Down
ALTER TABLE urls
DROP COLUMN visits;

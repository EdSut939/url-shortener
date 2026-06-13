-- +goose Up
CREATE TABLE urls (
    id int not null,
    short_url text,
    long_url text,
    primary key(id)
);

-- +goose Down
drop table urls
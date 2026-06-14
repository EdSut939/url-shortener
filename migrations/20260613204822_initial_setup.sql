-- +goose Up
CREATE TABLE urls (                                                                                                                                                                                                                        
    id int generated always as identity primary key,
    short_url varchar(30) not null unique,
    long_url text not null,
    ttl bigint,
    created_at timestamp default (timezone('utc', now()))
);

-- +goose Down
drop table urls;
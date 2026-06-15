-- +goose Up
CREATE TABLE urls (                                                                                                                                                                                                                        
    id int generated always as identity primary key,
    short_code varchar(10) not null unique,
    original_url text not null,
    ttl bigint,
    created_at timestamp default (timezone('utc', now()))
);

-- +goose Down
drop table urls;
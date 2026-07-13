-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id         serial primary key,
    email      varchar(150)                not null,
    firstname  varchar(150)                not null,
    lastname   varchar(150)                not null,
    password   varchar(255)                not null,
    role       varchar(20) default 'USER'  not null,
    created_at timestamp   default current_timestamp,
    updated_at timestamp   default current_timestamp,
    constraint users_email_uq unique (email)
);

CREATE TABLE IF NOT EXISTS profiles (
    id             bigint primary key,
    bio            text NULL,
    phone_number   varchar(15) NULL,
    date_of_birth  date NULL,
    constraint profiles_users_fk foreign key (id) references users (id) on delete cascade
);

-- +goose Down
DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS users;
-- +goose Up
CREATE TABLE IF NOT EXISTS users
(
    id         uuid  primary key DEFAULT gen_random_uuid(),
    email      varchar(150)     not null,
    username   varchar(150)     not null,
    password   varchar(255)     not null,
    role       varchar(20) default 'USER'  not null,
    created_at timestamp   default current_timestamp,
    updated_at timestamp   default current_timestamp,
    constraint users_email_uq unique (email),
	constraint users_username_uq unique (username)
);

CREATE TABLE IF NOT EXISTS profiles
(
    id             uuid primary key,
	firstname      varchar(150)     not null,
    lastname       varchar(150)     not null,
    bio            text             null,
    phone_number   varchar(15)      null,
    date_of_birth  date             null,
    constraint profiles_users_fk foreign key (id) references users (id) on delete cascade
);

-- +goose Down
DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS users;

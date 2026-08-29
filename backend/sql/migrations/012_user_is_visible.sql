-- +goose Up

-- +goose StatementBegin
CREATE FUNCTION user_is_visible(user_id uuid) RETURNS boolean
LANGUAGE sql STABLE AS $$
    SELECT deleted_at IS NULL AND suspended_at IS NULL
    FROM users
    WHERE users.id = user_id
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS user_is_visible(uuid);

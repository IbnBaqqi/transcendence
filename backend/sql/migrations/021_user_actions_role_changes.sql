-- +goose Up
ALTER TABLE user_actions DROP CONSTRAINT user_actions_action_check;
ALTER TABLE user_actions ADD CONSTRAINT user_actions_action_check
    CHECK (action IN ('suspended', 'reinstated', 'deleted', 'promoted', 'demoted'));

-- +goose Down
-- The rows are deleted rather than mapped: 'suspended', 'reinstated' and
-- 'deleted' have no honest equivalent for a role change, and a row that lies
-- about what happened is worse than one that is gone. They only exist because
-- of the feature this migration is undoing.
ALTER TABLE user_actions DROP CONSTRAINT user_actions_action_check;
DELETE FROM user_actions WHERE action IN ('promoted', 'demoted');
ALTER TABLE user_actions ADD CONSTRAINT user_actions_action_check
    CHECK (action IN ('suspended', 'reinstated', 'deleted'));

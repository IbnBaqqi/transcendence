package service

import "github.com/IbnBaqqi/transcendence/internal/database"

// isActive is the Go counterpart to the user_is_visible SQL function: a user is
// visible to other people only while neither inactive state applies. Both have
// to agree, which is why neither open-codes the two columns.
//
// Deliberately not used by the admin service: acting on a suspended account is
// how reinstatement works, so those paths check deleted_at alone.
func isActive(u database.User) bool {
	return !u.DeletedAt.Valid && !u.SuspendedAt.Valid
}

package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

// ChangePassword sets a new password for a caller who already has a session,
// ends every session and API key the account has, and returns the refresh
// token for a fresh session so the caller stays signed in.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, current, next string) (string, error) {
	user, err := s.db.GetUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("change password: look up user: %w", err)
	}

	// Checked before the compare: an account created through a provider has no
	// password, and this must not become the endpoint that gives it one. That
	// is the same rule RequestReset applies, for the same reason - a provider
	// address can live in a mailbox weaker than the provider account.
	if !user.Password.Valid {
		return "", &ConflictError{
			Message: "This account signs in with a provider, so there is no password to change",
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password.String), []byte(current)); err != nil {
		return "", &ForbiddenError{Message: "Current password is incorrect"}
	}

	if err := validatePassword(next); err != nil {
		return "", err
	}

	// A change to the same value is a no-op that still ends every other
	// session, which is a lot of consequence for nothing happening.
	if current == next {
		return "", &ValidationError{Message: "New password must be different from the current one"}
	}

	// Outside the transaction on purpose - bcrypt would hold the row locks open
	// for the whole hash.
	hashed, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("change password: hash password: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("change password: begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("password change transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	if err := qtx.UpdateUserPassword(ctx, database.UpdateUserPasswordParams{
		ID:       userID,
		Password: sql.NullString{String: string(hashed), Valid: true},
	}); err != nil {
		return "", fmt.Errorf("change password: update password: %w", err)
	}

	if err := qtx.RevokeSessionsForPasswordChange(ctx, userID); err != nil {
		return "", fmt.Errorf("change password: revoke sessions: %w", err)
	}

	if err := qtx.RevokeKeysForUser(ctx, userID); err != nil {
		return "", fmt.Errorf("change password: revoke api keys: %w", err)
	}

	// After the revocation, never before it: a session issued first would
	// revoke itself and log the caller out with their own request.
	refreshToken, err := s.IssueSession(ctx, qtx, userID)
	if err != nil {
		return "", fmt.Errorf("change password: store session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("change password: commit: %w", err)
	}

	return refreshToken, nil
}

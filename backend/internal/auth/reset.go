package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/notify"
)

const (
	resetTokenTTL = time.Hour
	resetCooldown = 2 * time.Minute

	resetPath = "/reset-password"
)

const resetFailedMessage = "invalid or expired reset link"

func (s *Service) RequestReset(ctx context.Context, email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return &ValidationError{Message: "email is required"}
	}

	user, err := s.db.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("password reset: look up email: %w", err)
	}

	raw := MakeRefreshToken()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("password reset: begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("password reset transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	if _, err := qtx.GetUserForUpdate(ctx, user.ID); err != nil {
		return fmt.Errorf("password reset: lock user: %w", err)
	}

	last, err := qtx.LastResetRequestAt(ctx, user.ID)
	switch {
	case err == nil:
		if time.Since(last) < resetCooldown {
			slog.Info("password reset request ignored, inside cooldown", "user_id", user.ID)
			return nil
		}
	case errors.Is(err, sql.ErrNoRows):
	default:
		return fmt.Errorf("password reset: check cooldown: %w", err)
	}

	// Supersede any outstanding link, so asking twice leaves one key, not two.
	if err := qtx.InvalidateResetTokensForUser(ctx, user.ID); err != nil {
		return fmt.Errorf("password reset: invalidate previous: %w", err)
	}

	if err := qtx.CreateResetToken(ctx, database.CreateResetTokenParams{
		TokenHash: hashSession(raw),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(resetTokenTTL),
	}); err != nil {
		return fmt.Errorf("password reset: store token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("password reset: commit: %w", err)
	}

	s.notify.Notify(ctx, notify.PasswordReset(user.Email, s.resetLink(raw), resetTokenTTL))

	return nil
}

func (s *Service) resetLink(rawToken string) string {
	return strings.TrimSuffix(s.frontendURL, "/") + resetPath + "?token=" + url.QueryEscape(rawToken)
}

// ResetPassword redeems a link: sets the new password, spends the token, and
// ends every existing session.
func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if rawToken == "" {
		return &AuthError{Message: resetFailedMessage}
	}
	if err := validatePassword(newPassword); err != nil {
		return err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("password reset: hash password: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("password reset: begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("password reset transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	row, err := qtx.FindLiveResetToken(ctx, hashSession(rawToken))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Unknown, expired and already spent share a message on purpose.
			return &AuthError{Message: resetFailedMessage}
		}
		return fmt.Errorf("password reset: find token: %w", err)
	}

	// Find-then-update is check-then-act: two requests redeeming the same link
	// both reach here, and the row count is what makes one of them lose.
	spent, err := qtx.MarkResetTokenUsed(ctx, row.TokenHash)
	if err != nil {
		return fmt.Errorf("password reset: spend token: %w", err)
	}
	if spent == 0 {
		return &AuthError{Message: resetFailedMessage}
	}

	if err := qtx.UpdateUserPassword(ctx, database.UpdateUserPasswordParams{
		ID:       row.UserID,
		Password: string(hashed),
	}); err != nil {
		return fmt.Errorf("password reset: update password: %w", err)
	}

	// The point of the flow: you reset because someone else may have your
	// password, so their access must not outlive it.
	if err := qtx.RevokeSessionsForPasswordReset(ctx, row.UserID); err != nil {
		return fmt.Errorf("password reset: revoke sessions: %w", err)
	}

	if err := qtx.RevokeKeysForUser(ctx, row.UserID); err != nil {
		return fmt.Errorf("password reset: revoke api keys: %w", err)
	}

	return tx.Commit()
}

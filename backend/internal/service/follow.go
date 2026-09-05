package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"slices"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

const (
	followeeConstraint = "follows_followee_id_fkey"
	followerConstraint = "follows_follower_id_fkey"
)

type FollowService struct {
	db *database.DB
}

func NewFollowService(db *database.DB) *FollowService {
	return &FollowService{db: db}
}

func (s *FollowService) Follow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	if followerID == followeeID {
		return &ValidationError{Message: "You cannot follow yourself"}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("follow transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	inserted, err := qtx.FollowUser(ctx, database.FollowUserParams{
		FollowerID: followerID,
		FolloweeID: followeeID,
	})
	if isForeignKeyViolation(err, followeeConstraint, followerConstraint) {
		return &NotFoundError{Message: "User not found"}
	}
	if err != nil {
		return err
	}
	if inserted == 0 {
		return nil
	}

	if err := recordActorNotification(ctx, qtx, followeeID, notifyKindNewFollower, followerID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *FollowService) Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	_, err := s.db.UnfollowUser(ctx, database.UnfollowUserParams{
		FollowerID: followerID,
		FolloweeID: followeeID,
	})
	return err
}

// viewerID decides whose blocks hide presence in the result - it is the person
// reading the list, not the person the list is about.
func (s *FollowService) ListFollowing(ctx context.Context, viewerID, subjectID uuid.UUID) ([]database.ListFollowingRow, error) {
	if err := s.requireUser(ctx, subjectID); err != nil {
		return nil, err
	}
	return s.db.ListFollowing(ctx, database.ListFollowingParams{
		ViewerID:  viewerID,
		SubjectID: subjectID,
	})
}

func (s *FollowService) ListFollowers(ctx context.Context, viewerID, subjectID uuid.UUID) ([]database.ListFollowersRow, error) {
	if err := s.requireUser(ctx, subjectID); err != nil {
		return nil, err
	}
	return s.db.ListFollowers(ctx, database.ListFollowersParams{
		ViewerID:  viewerID,
		SubjectID: subjectID,
	})
}

func (s *FollowService) requireUser(ctx context.Context, userID uuid.UUID) error {
	if _, err := s.db.GetUser(ctx, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &NotFoundError{Message: "User not found"}
		}
		return err
	}
	return nil
}

func isForeignKeyViolation(err error, constraints ...string) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code != "23503" {
		return false
	}
	return slices.Contains(constraints, pqErr.Constraint)
}

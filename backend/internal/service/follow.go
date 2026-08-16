package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

const followeeConstraint = "follows_followee_id_fkey"

type FollowService struct {
	db *database.Queries
}

func NewFollowService(db *database.Queries) *FollowService {
	return &FollowService{db: db}
}

func (s *FollowService) Follow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	if followerID == followeeID {
		return &ValidationError{Message: "you cannot follow yourself"}
	}

	err := s.db.FollowUser(ctx, database.FollowUserParams{
		FollowerID: followerID,
		FolloweeID: followeeID,
	})
	if isForeignKeyViolation(err, followeeConstraint) {
		return &NotFoundError{Message: "user not found"}
	}
	return err
}

func (s *FollowService) Unfollow(ctx context.Context, followerID, followeeID uuid.UUID) error {
	rows, err := s.db.UnfollowUser(ctx, database.UnfollowUserParams{
		FollowerID: followerID,
		FolloweeID: followeeID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return &NotFoundError{Message: "you are not following this user"}
	}
	return nil
}

func (s *FollowService) ListFollowing(ctx context.Context, userID uuid.UUID) ([]database.ListFollowingRow, error) {
	return s.db.ListFollowing(ctx, userID)
}

func (s *FollowService) ListFollowers(ctx context.Context, userID uuid.UUID) ([]database.ListFollowersRow, error) {
	if _, err := s.db.GetUser(ctx, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &NotFoundError{Message: "user not found"}
		}
		return nil, err
	}
	return s.db.ListFollowers(ctx, userID)
}

func isForeignKeyViolation(err error, constraint string) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}
	return pqErr.Code == "23503" && pqErr.Constraint == constraint
}

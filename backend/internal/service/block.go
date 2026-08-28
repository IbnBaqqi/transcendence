package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

const (
	blockerConstraint = "blocks_blocker_id_fkey"
	blockedConstraint = "blocks_blocked_id_fkey"
)

type BlockService struct {
	db *database.Queries
}

func NewBlockService(db *database.Queries) *BlockService {
	return &BlockService{db: db}
}

func (s *BlockService) Block(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	if blockerID == blockedID {
		return &ValidationError{Message: "You cannot block yourself"}
	}

	err := s.db.BlockUser(ctx, database.BlockUserParams{
		BlockerID: blockerID,
		BlockedID: blockedID,
	})
	if isForeignKeyViolation(err, blockerConstraint, blockedConstraint) {
		return &NotFoundError{Message: "User not found"}
	}
	return err
}

// ExistsBetween reports whether either user has blocked the other. Presence is
// hidden on a block in either direction, so callers do not care who blocked whom.
func (s *BlockService) ExistsBetween(ctx context.Context, a, b uuid.UUID) (bool, error) {
	return s.db.BlockExistsBetween(ctx, database.BlockExistsBetweenParams{UserA: a, UserB: b})
}

func (s *BlockService) Unblock(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	_, err := s.db.UnblockUser(ctx, database.UnblockUserParams{
		BlockerID: blockerID,
		BlockedID: blockedID,
	})
	return err
}

func (s *BlockService) List(ctx context.Context, userID uuid.UUID) ([]database.ListBlocksRow, error) {
	return s.db.ListBlocks(ctx, userID)
}

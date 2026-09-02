package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

type TagService struct {
	db *database.DB
}

func NewTagService(db *database.DB) *TagService {
	return &TagService{db: db}
}

func (s *TagService) SweepUnused(ctx context.Context) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("tag sweep rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	if err := qtx.LockTagsForSweep(ctx); err != nil {
		return 0, err
	}

	deleted, err := qtx.DeleteUnusedTags(ctx)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return deleted, nil
}

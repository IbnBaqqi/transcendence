package main

import (
	"context"
	"fmt"
	"os"

	"github.com/IbnBaqqi/transcendence/internal/config"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/logger"
	"github.com/IbnBaqqi/transcendence/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "tag sweep failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	log, err := logger.New(cfg.Logger.Level, cfg.Env)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	ctx := context.Background()

	db, err := database.Connect(ctx, &cfg.DB)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := database.AssertSchemaCurrent(ctx, db.DB); err != nil {
		return err
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Error("failed to close database connection", "error", err)
		}
	}()

	deleted, err := service.NewTagService(db).SweepUnused(ctx)
	if err != nil {
		return fmt.Errorf("failed to sweep unused tags: %w", err)
	}

	fmt.Printf("deleted %d unused tag(s)\n", deleted)

	return nil
}

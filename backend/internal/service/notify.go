package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/notify"
)

func notifyUser(
	ctx context.Context,
	q *database.Queries,
	n notify.Notifier,
	userID uuid.UUID,
	build func(email, username string) notify.Message,
) {
	user, err := q.GetUser(ctx, userID)
	if err != nil {
		slog.Error("could not look up a notification recipient",
			"user_id", userID, "error", err)
		return
	}
	n.Notify(ctx, build(user.Email, user.Username))
}

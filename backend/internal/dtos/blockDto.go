package dtos

import (
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

type BlockedUserResponse struct {
	ID        uuid.UUID  `json:"id"`
	Username  string     `json:"username"`
	BlockedAt *time.Time `json:"blocked_at"`
}

func ToBlockedUserResponses(rows []database.ListBlocksRow) []BlockedUserResponse {
	out := make([]BlockedUserResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, BlockedUserResponse{
			ID:        r.ID,
			Username:  r.Username,
			BlockedAt: nullTimePtr(r.CreatedAt),
		})
	}
	return out
}
